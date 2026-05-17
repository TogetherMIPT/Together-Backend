import os
import gc
import torch
import time
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
from transformers import AutoModelForCausalLM, AutoTokenizer
from peft import PeftModel
from huggingface_hub import login
from tqdm import tqdm
import warnings

warnings.filterwarnings("ignore")

HF_TOKEN = os.getenv("HF_TOKEN") or input("HF Token (repo.read): ")
try:
    login(token=HF_TOKEN, add_to_git_credential=True)
except Exception as e:
    print(f"Ошибка авторизации: {e}")
    exit(1)

BASE_MODEL = "sberbank-ai/rugpt3medium_based_on_gpt2"
ADAPTER_REPO = "nikrog/rugpt3medium_finetuned_psychology"
DEVICE = "cuda" if torch.cuda.is_available() else "cpu"
PROMPT = "Клиент испытывает тревожность перед экзаменами. Какие техники релаксации можно порекомендовать?"

print("Загрузка базовой модели...")
base_model = AutoModelForCausalLM.from_pretrained(
    BASE_MODEL,
    device_map="auto",
    torch_dtype=torch.float16 if DEVICE == "cuda" else torch.float32,
    token=HF_TOKEN,
    low_cpu_mem_usage=True
)

print("Загрузка адаптера и слияние...")
peft_model = PeftModel.from_pretrained(base_model, ADAPTER_REPO, token=HF_TOKEN)
model = peft_model.merge_and_unload()
model.eval()

# Очистка памяти после слияния
if DEVICE == "cuda":
    gc.collect()
    torch.cuda.empty_cache()

tokenizer = AutoTokenizer.from_pretrained(ADAPTER_REPO, token=HF_TOKEN)
if tokenizer.pad_token is None:
    tokenizer.pad_token = tokenizer.eos_token

print(f"Модель готова на {DEVICE}. VRAM: {torch.cuda.memory_allocated()/1e9:.2f} GB")

# Функция замера
def measure_generation(**gen_kwargs):
    inputs = tokenizer(PROMPT, return_tensors="pt").to(DEVICE)
    input_len = inputs["input_ids"].shape[1]
    
    if DEVICE == "cuda":
        torch.cuda.synchronize()
        
    start = time.perf_counter()
    with torch.no_grad():
        output = model.generate(
            **inputs,
            **gen_kwargs,
            pad_token_id=tokenizer.eos_token_id,
            return_dict_in_generate=True,
            output_scores=False
        )
    if DEVICE == "cuda":
        torch.cuda.synchronize()
        
    total_time = time.perf_counter() - start
    generated_tokens = output.sequences.shape[1] - input_len
    
    return {
        "total_time_sec": round(total_time, 4),
        "generated_tokens": generated_tokens,
        "tokens_per_sec": round(generated_tokens / total_time, 2) if total_time > 0 else 0
    }

# Сетка параметров
param_grid = {
    "max_new_tokens": [50, 100, 200],
    "num_beams": [1, 2],
    "do_sample": [False, True],
    "temperature": [0.7, 1.0],
    "top_k": [None, 50]
}

results = []
print("\nЗапуск экспериментов...")
total_runs = (len(param_grid["max_new_tokens"]) * len(param_grid["num_beams"]) * 
              len(param_grid["do_sample"]) * len(param_grid["temperature"]) * len(param_grid["top_k"]))

with tqdm(total=total_runs, desc="Прогресс") as pbar:
    for max_new_tokens in param_grid["max_new_tokens"]:
        for num_beams in param_grid["num_beams"]:
            for do_sample in param_grid["do_sample"]:
                for temperature in ([1.0] if not do_sample else param_grid["temperature"]):
                    for top_k in param_grid["top_k"]:
                        if top_k is not None and not do_sample:
                            pbar.update(1)
                            continue
                            
                        config = {
                            "max_new_tokens": max_new_tokens,
                            "num_beams": num_beams,
                            "do_sample": do_sample,
                            "temperature": temperature,
                            "use_cache": True
                        }
                        if top_k is not None:
                            config["top_k"] = top_k
                            
                        try:
                            metrics = measure_generation(**config)
                            results.append({**config, **metrics})
                            pbar.set_postfix(tok_s=metrics["tokens_per_sec"])
                        except Exception as e:
                            print(f"\nОшибка при {config}: {e}")
                        pbar.update(1)

df = pd.DataFrame(results)
df.to_csv("rugpt3medium_peft_benchmark.csv", index=False)
print(f"\nГотово! {len(df)} комбинаций сохранено.")

sns.set_theme(style="whitegrid", font_scale=1.1)
plt.rcParams["figure.dpi"] = 120

# 1. Зависимость скорости от длины и beam search
plt.figure(figsize=(10, 5))
sns.lineplot(data=df, x="max_new_tokens", y="tokens_per_sec", 
             hue="num_beams", style="do_sample", marker="o", linewidth=2)
plt.title("Токенов/сек vs max_new_tokens и num_beams", fontsize=14)
plt.xlabel("max_new_tokens"); plt.ylabel("Токенов/сек")
plt.legend(bbox_to_anchor=(1.02, 1), loc='upper left')
plt.tight_layout()
plt.savefig("speed_vs_maxtokens_beams.png", dpi=300, bbox_inches='tight')
plt.show()

# 2. Влияние сэмплирования на скорость
plt.figure(figsize=(6, 4))
sns.boxplot(data=df[df["max_new_tokens"]==100], 
            x="do_sample", y="tokens_per_sec", hue="temperature")
plt.title("do_sample и temperature (max_new_tokens=100)")
plt.ylabel("Токенов/сек")
plt.tight_layout()
plt.savefig("sample_vs_greedy.png", dpi=300, bbox_inches='tight')
plt.show()
