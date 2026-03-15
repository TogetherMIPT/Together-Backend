import torch
from transformers import (
    AutoTokenizer,
    AutoModelForCausalLM,
    TrainingArguments,
    Trainer,
    DataCollatorForLanguageModeling,
    BitsAndBytesConfig
)
from peft import (
    LoraConfig,
    get_peft_model,
    prepare_model_for_kbit_training,
    TaskType
)
from datasets import load_dataset
import os
from huggingface_hub import login

# Конфигурация
MODEL_NAME = "sberbank-ai/rugpt3medium_based_on_gpt2"  # базовая модель для дообучения ruGPT3
DATASET_NAME = "nikrog/psychology_dataset_rus"  # датасет
FINETUNED_MODEL_NAME = "nikrog/rugpt3medium_finetuned_psychology"
OUTPUT_DIR = "./rugpt3_finetuned"

# Параметры LoRA/QLoRA
LORA_R = 16
LORA_ALPHA = 32
LORA_DROPOUT = 0.05
TARGET_MODULES = ["c_attn"]  # для GPT-2 архитектуры

# QLoRA настройки
USE_QLORA = True  # True для QLoRA, False для обычного LoRA
LOAD_IN_4BIT = True
LOAD_IN_8BIT = False

def load_model_and_tokenizer():
    """Загрузка модели и токенизатора"""
    tokenizer = AutoTokenizer.from_pretrained(MODEL_NAME)
    tokenizer.pad_token = tokenizer.eos_token
    
    if USE_QLORA:
        # Конфигурация для 4-битного квантования
        quantization_config = BitsAndBytesConfig(
            load_in_4bit=LOAD_IN_4BIT,
            load_in_8bit=LOAD_IN_8BIT,
            bnb_4bit_compute_dtype=torch.float16,
            bnb_4bit_quant_type="nf4",
            bnb_4bit_use_double_quant=True,
        )
        
        model = AutoModelForCausalLM.from_pretrained(
            MODEL_NAME,
            quantization_config=quantization_config,
            device_map="auto",
            trust_remote_code=True
        )
        
        # Подготовка модели для k-bit training
        model = prepare_model_for_kbit_training(model)
    else:
        # Обычная загрузка для LoRA
        model = AutoModelForCausalLM.from_pretrained(
            MODEL_NAME,
            device_map="auto",
            torch_dtype=torch.float16
        )
    
    return model, tokenizer

def configure_lora(model):
    """Настройка LoRA адаптеров"""
    lora_config = LoraConfig(
        r=LORA_R,
        lora_alpha=LORA_ALPHA,
        target_modules=TARGET_MODULES,
        lora_dropout=LORA_DROPOUT,
        bias="none",
        task_type=TaskType.CAUSAL_LM,
    )
    
    model = get_peft_model(model, lora_config)
    model.print_trainable_parameters()
    
    return model


def load_and_preprocess_dataset(tokenizer, max_length=512):
    """Загрузка и предобработка датасета"""

    hf_token = "MY_TOKEN"

    login(token=hf_token)
    print("Success authentification in Huggingface Hub")

    data_files = {
        "train": "train.txt",
        "test": "test.txt",
        "val": "val.txt"
    }
    
    # Загрузка датасета из Hugging Face
    dataset = load_dataset(DATASET_NAME, data_files=data_files, token=hf_token)
    dataset_train = dataset["train"]
    dataset_val = dataset["val"]
    dataset_test = dataset["test"]
    
    def preprocess_function(examples):
        # Токенизация текста
        tokenized = tokenizer(
            examples["text"],
            truncation=True,
            max_length=max_length,
            padding="max_length"
        )
        tokenized["labels"] = tokenized["input_ids"].copy()
        return tokenized
    
    # Применение предобработки
    processed_train_dataset = dataset_train.map(
        preprocess_function,
        batched=True,
        remove_columns=dataset_train.column_names
    )

    processed_val_dataset = dataset_val.map(
        preprocess_function,
        batched=True,
        remove_columns=dataset_val.column_names
    )

    processed_test_dataset = dataset_test.map(
        preprocess_function,
        batched=True,
        remove_columns=dataset_test.column_names
    )
    
    # Разделение на train/eval
    #split_dataset = processed_dataset.train_test_split(test_size=0.1)
    #return split_dataset["train"], split_dataset["test"]

    return processed_train_dataset, processed_val_dataset, processed_test_dataset

def train_model(model, tokenizer, train_dataset, eval_dataset):
    """Обучение модели"""
    # Data collator
    data_collator = DataCollatorForLanguageModeling(
        tokenizer=tokenizer,
        mlm=False
    )
    
    # Параметры обучения
    training_args = TrainingArguments(
        output_dir=OUTPUT_DIR,
        per_device_train_batch_size=4,
        per_device_eval_batch_size=4,
        gradient_accumulation_steps=4,
        learning_rate=2e-4,
        num_train_epochs=3,
        weight_decay=0.01,
        fp16=True,
        logging_steps=10,
        save_strategy="epoch",
        eval_strategy="epoch",
        load_best_model_at_end=True,
        report_to="none",
        gradient_checkpointing=True if USE_QLORA else False,
    )
    
    # Инициализация Trainer
    trainer = Trainer(
        model=model,
        args=training_args,
        train_dataset=train_dataset,
        eval_dataset=eval_dataset,
        data_collator=data_collator,
    )
    
    # Запуск обучения
    trainer.train()
    
    # Сохранение модели
    trainer.save_model(OUTPUT_DIR)
    tokenizer.save_pretrained(OUTPUT_DIR)
    
    return trainer


def save_model_to_hf(trainer, tokenizer):
    """Сохранение модели"""
    # Загрузка на Hugging Face
    trainer.push_to_hub(
        repo_id=FINETUNED_MODEL_NAME,
        private=True,
        commit_message="Load fine-tuned model on psychology dataset",
    )

    # Токенизатор тоже нужно загрузить
    tokenizer.push_to_hub(
        repo_id=FINETUNED_MODEL_NAME,
        private=True,
        commit_message="Load tokenizer for fine-tuned model"
    )

def main():
    print("Загрузка модели и токенизатора...")
    model, tokenizer = load_model_and_tokenizer()
    
    print("Настройка LoRA адаптеров...")
    model = configure_lora(model)
    
    print("Загрузка и предобработка датасета...")
    train_dataset, eval_dataset, test_dataset = load_and_preprocess_dataset(tokenizer)
    
    print("Начало обучения...")
    trainer = train_model(model, tokenizer, train_dataset, eval_dataset)
    
    print("Обучение завершено!")
    print(f"Модель сохранена в: {OUTPUT_DIR}")

if __name__ == "__main__":
    main()