import torch
import time
import gc
import os
import json
import yaml
import pandas as pd
import numpy as np
from transformers import (
    AutoTokenizer, AutoModelForCausalLM, TrainingArguments, Trainer,
    DataCollatorForLanguageModeling, BitsAndBytesConfig
)
from peft import LoraConfig, get_peft_model, prepare_model_for_kbit_training, TaskType
from datasets import load_dataset

# ==========================================
# КОНФИГУРАЦИЯ ИССЛЕДОВАНИЯ
# ==========================================
BASE_MODEL = "sberbank-ai/rugpt3medium_based_on_gpt2"
DATASET_ID = "nikrog/psychology_dataset_rus"
RESULTS_FILE = "lora_hyperparameter_study.csv"
RESPONSES_FILE = "psychology_case_responses.jsonl"  # Новый файл для ответов моделей
CASES_YAML_PATH = "test_cases.yaml"                 # Путь к YAML с тестовыми случаями
MAX_TRAIN_SAMPLES = 2000
MAX_EVAL_SAMPLES = 200
MAX_SEQ_LENGTH = 512

# Промпт для генерации (адаптирован под стиль GPT-2/RuGPT3)
PROMPT_TEMPLATE = "Ситуация: {situation}\nРекомендация психолога:"

# Сетка гиперпараметров
# HP_GRID = [
#     {"r": 8,  "alpha": 16, "bits": 4, "epochs": 1, "lr": 1e-4},
#     {"r": 8,  "alpha": 16, "bits": 4, "epochs": 1, "lr": 2e-4},
#     {"r": 16, "alpha": 32, "bits": 4, "epochs": 1, "lr": 1e-4},
#     {"r": 16, "alpha": 32, "bits": 4, "epochs": 1, "lr": 2e-4},
#     {"r": 32, "alpha": 64, "bits": 4, "epochs": 1, "lr": 1e-4},
#     {"r": 32, "alpha": 64, "bits": 4, "epochs": 1, "lr": 2e-4},
# ]

HP_GRID = [
    # =====================================================================
    # СЕКЦИЯ: Влияние Learning Rate (фиксированные r=16, alpha=32)
    # Цель: найти оптимальную скорость обучения для базовой конфигурации
    # =====================================================================
    {"r": 16, "alpha": 32, "bits": 4, "epochs": 1, "lr": 1e-4},  # Консервативный
    {"r": 16, "alpha": 32, "bits": 4, "epochs": 1, "lr": 2e-4},  # Базовый (стандарт)
    {"r": 16, "alpha": 32, "bits": 4, "epochs": 1, "lr": 3e-4},  # Агрессивный (новый!)
    
    # =====================================================================
    # СЕКЦИЯ: Влияние соотношения α/r (фиксированные r=16, lr=2e-4)
    # Цель: понять, важен ли абсолютный alpha или только отношение α/r
    # =====================================================================
    {"r": 16, "alpha": 16, "bits": 4, "epochs": 1, "lr": 2e-4},  # α/r = 1.0 (слабое обновление)
    # {"r": 16, "alpha": 32, "bits": 4, "epochs": 1, "lr": 2e-4},  # α/r = 2.0 (уже есть в секции 1)
    {"r": 16, "alpha": 64, "bits": 4, "epochs": 1, "lr": 2e-4},  # α/r = 4.0 (сильное обновление)
    
    # =====================================================================
    # СЕКЦИЯ: Влияние ранга r (фиксированные α=2r, lr=2e-4)
    # Цель: оценить, даёт ли увеличение ёмкости адаптера прирост качества
    # =====================================================================
    {"r": 8,  "alpha": 16, "bits": 4, "epochs": 1, "lr": 2e-4},  # Малый ранг (экономный)
    # {"r": 16, "alpha": 32, "bits": 4, "epochs": 1, "lr": 2e-4}, # Средний (уже есть)
    {"r": 32, "alpha": 64, "bits": 4, "epochs": 1, "lr": 2e-4},  # Большой ранг (ёмкий)
    
    # =====================================================================
    # СЕКЦИЯ: Контрольные точки (опционально, для проверки взаимодействий)
    # =====================================================================
    # {"r": 8,  "alpha": 8,  "bits": 4, "epochs": 1, "lr": 1e-4},  # Минимальная конфигурация
    # {"r": 32, "alpha": 128,"bits": 4, "epochs": 1, "lr": 3e-4},  # Максимальная конфигурация
]

def load_test_cases(yaml_path):
    """Загрузка тестовых случаев из YAML"""
    if not os.path.exists(yaml_path):
        raise FileNotFoundError(f"Файл {yaml_path} не найден. Создайте его перед запуском.")
    with open(yaml_path, 'r', encoding='utf-8') as f:
        data = yaml.safe_load(f)
    return data.get('cases', [])

def load_and_prepare_data(tokenizer):
    """Загрузка и токенизация датасета"""
    print("Загрузка датасета...")
    dataset = load_dataset(DATASET_ID)
    train_ds = dataset["train"].select(range(min(len(dataset["train"]), MAX_TRAIN_SAMPLES)))
    eval_ds = dataset["val"].select(range(min(len(dataset["val"]), MAX_EVAL_SAMPLES)))
    
    def tokenize_fn(examples):
        return tokenizer(examples["text"], truncation=True, max_length=MAX_SEQ_LENGTH)
    
    train_ds = train_ds.map(tokenize_fn, batched=True, remove_columns=train_ds.column_names)
    eval_ds = eval_ds.map(tokenize_fn, batched=True, remove_columns=eval_ds.column_names)
    return train_ds, eval_ds

def generate_case_responses(model, tokenizer, cases, max_new_tokens=256):
    """Генерация ответов на тестовые ситуации"""
    model.eval()
    responses = []
    print(f"Генерация ответов на {len(cases)} случаев...")
    
    for case in cases:
        situation = case["situation"].strip()
        prompt = PROMPT_TEMPLATE.format(situation=situation)
        
        inputs = tokenizer(
            prompt, return_tensors="pt", truncation=True, max_length=MAX_SEQ_LENGTH
        ).to(model.device)
        prompt_len = inputs.input_ids.shape[1]
        
        with torch.inference_mode():
            outputs = model.generate(
                inputs.input_ids,
                max_new_tokens=max_new_tokens,
                do_sample=True,
                temperature=0.7,
                top_p=0.9,
                repetition_penalty=1.1,
                pad_token_id=tokenizer.eos_token_id
            )
        
        full_text = tokenizer.decode(outputs[0], skip_special_tokens=True)
        generated = full_text[len(prompt):].strip()
        
        responses.append({
            "case_id": case["id"],
            "case_title": case["title"],
            "situation": situation,
            "generated_answer": generated,
            "reference_answer": case.get("reference_answer", ""),
            "evaluation_criteria": case.get("evaluation_criteria", {})
        })
    return responses

def run_experiment(cfg, tokenizer, train_ds, eval_ds, run_idx, test_cases):
    """Один запуск эксперимента"""
    torch.cuda.empty_cache()
    gc.collect()
    
    r, alpha, bits, epochs, lr = cfg["r"], cfg["alpha"], cfg["bits"], cfg["epochs"], cfg["lr"]
    exp_name = f"run{run_idx}_r{r}_a{alpha}_b{bits}_e{epochs}_lr{lr:.0e}"
    output_dir = os.path.join("./hp_results", exp_name)
    
    print(f"\nЗапуск: {exp_name}")
    
    # 1. Квантизация
    bnb_config = BitsAndBytesConfig(
        load_in_4bit=True if bits == 4 else False,
        load_in_8bit=True if bits == 8 else False,
        bnb_4bit_compute_dtype=torch.float16,
        bnb_4bit_quant_type="nf4",
        bnb_4bit_use_double_quant=True
    ) if bits in [4, 8] else None
        
    model = AutoModelForCausalLM.from_pretrained(
        BASE_MODEL,
        quantization_config=bnb_config,
        device_map="auto",
        torch_dtype=torch.float16
    )
    if bits in [4, 8]:
        model = prepare_model_for_kbit_training(model)
    
    # 2. LoRA
    lora_config = LoraConfig(
        r=r, lora_alpha=alpha, target_modules=["c_attn", "c_proj"],
        lora_dropout=0.05, bias="none", task_type=TaskType.CAUSAL_LM
    )
    model = get_peft_model(model, lora_config)
    
    # 3. Trainer
    training_args = TrainingArguments(
        output_dir=output_dir,
        per_device_train_batch_size=2,
        gradient_accumulation_steps=4,
        learning_rate=lr,
        num_train_epochs=epochs,
        fp16=True,
        optim="paged_adamw_32bit",
        gradient_checkpointing=True,
        gradient_checkpointing_kwargs={"use_reentrant": False},
        eval_strategy="epoch",
        logging_steps=50,
        save_strategy="no",
        report_to="none",
        dataloader_num_workers=2,
        warmup_ratio=0.05,
        lr_scheduler_type="cosine"
    )
    
    collator = DataCollatorForLanguageModeling(tokenizer=tokenizer, mlm=False)
    trainer = Trainer(model=model, args=training_args, train_dataset=train_ds,
                      eval_dataset=eval_ds, data_collator=collator)
    
    # 4. Обучение
    start_time = time.time()
    trainer.train()
    train_time = time.time() - start_time
    
    # 5. Оценка
    metrics = trainer.evaluate()
    eval_loss = metrics.get("eval_loss", float("nan"))
    perplexity = np.exp(eval_loss) if not np.isnan(eval_loss) else float("nan")
    
    # 6. Сохранение адаптера
    trainer.save_model(output_dir)
    
    # 7. Генерация ответов на тестовые случаи
    print("Генерация ответов на психологические случаи...")
    try:
        case_responses = generate_case_responses(model, tokenizer, test_cases, max_new_tokens=256)
        with open(RESPONSES_FILE, 'a', encoding='utf-8') as f:
            for resp in case_responses:
                resp["run_idx"] = run_idx
                resp.update(cfg)  # Добавляем гиперпараметры для контекста
                f.write(json.dumps(resp, ensure_ascii=False) + "\n")
        print(f"Ответы сохранены в {RESPONSES_FILE}")
    except Exception as e:
        print(f"Ошибка при генерации ответов: {e}")
    
    # 8. Очистка
    del model, trainer, collator
    torch.cuda.empty_cache()
    gc.collect()
    
    return {
        "run_idx": run_idx,
        "r": r, "alpha": alpha, "bits": bits, "epochs": epochs, "lr": lr,
        "train_time_sec": round(train_time, 2),
        "eval_loss": round(eval_loss, 4),
        "perplexity": round(perplexity, 2),
        "status": "success"
    }

def main():
    print("Инициализация...")
    tokenizer = AutoTokenizer.from_pretrained(BASE_MODEL)
    tokenizer.pad_token = tokenizer.eos_token
    
    print("Загрузка тестовых случаев...")
    test_cases = load_test_cases(CASES_YAML_PATH)
    
    train_ds, eval_ds = load_and_prepare_data(tokenizer)
    
    results = []
    if os.path.exists(RESULTS_FILE):
        results = pd.read_csv(RESULTS_FILE).to_dict(orient="records")
        print(f"Загружено {len(results)} предыдущих записей.")
    
    start_run = len(results) + 1
    for i, cfg in enumerate(HP_GRID):
        run_idx = start_run + i
        try:
            res = run_experiment(cfg, tokenizer, train_ds, eval_ds, run_idx, test_cases)
            results.append(res)
            pd.DataFrame(results).to_csv(RESULTS_FILE, index=False)
            print(f"Готово. Perplexity: {res['perplexity']:.2f} | Время: {res['train_time_sec']/60:.1f} мин")
        except Exception as e:
            print(f"Ошибка в run {run_idx}: {e}")
            results.append({"run_idx": run_idx, "status": f"failed: {str(e)}", **cfg})
            pd.DataFrame(results).to_csv(RESULTS_FILE, index=False)
            
    print("\nИсследование завершено. Результаты в", RESULTS_FILE)
    print("Ответы моделей сохранены в", RESPONSES_FILE)

if __name__ == "__main__":
    main()
