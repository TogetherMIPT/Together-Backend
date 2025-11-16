import torch
import os
import sys
import json
from transformers import AutoModelForCausalLM, AutoTokenizer
import numpy as np
from pathlib import Path
from huggingface_hub import login


def get_hf_token():
    token = os.getenv('HF_TOKEN')
    if not token:
        raise ValueError("HF_TOKEN not found in environment variables")
    
    return token

def load_model_and_tokenizer_from_hf(model_name):
    try:
        hf_token = get_hf_token()
        
        login(token=hf_token)
        print("Success authentification in Huggingface Hub")

        model = AutoModelForCausalLM.from_pretrained(
            model_name,
            trust_remote_code=True
        )
        tokenizer = AutoTokenizer.from_pretrained(
            model_name,
            trust_remote_code=True
        )
        
        if tokenizer.pad_token is None:
            tokenizer.pad_token = tokenizer.eos_token
            
        print("Success load of model and tokenizer!")
        return model, tokenizer
        
    except Exception as e:
        print(f"Model Load Error: {e}")
        raise

def calculate_perplexity(model, tokenizer, text_data, max_length=1024):
    losses = []
    for text in text_data:
        # Токенизация с добавлением EOS токена
        inputs = tokenizer.encode(text + tokenizer.eos_token, 
                                return_tensors='pt',
                                max_length=max_length,
                                truncation=True)

        with torch.no_grad():
            outputs = model(inputs, labels=inputs)
            loss = outputs.loss
            losses.append(loss.item())

    perplexity = np.exp(np.mean(losses))
    return perplexity


def load_dataset_from_hf(dataset_name, split="testv2", text_column="text"):
    try:
        hf_token = get_hf_token()

        login(token=hf_token)
        print("Success authentification in Huggingface Hub")
        
        dataset = load_dataset(dataset_name)
        
        available_splits = list(dataset.keys())
        print(f"Available files: {available_splits}")
        
        if split not in available_splits:
            raise(f"File {split} not found")
        
        split_data = dataset[split]
        
        # Извлекаем тексты
        if text_column in split_data.column_names:
            texts = [item for item in split_data[text_column] if item and str(item).strip()]
        else:
            available_cols = split_data.column_names
            text_cols = [col for col in available_cols if isinstance(split_data[0][col], str)]
            if text_cols:
                text_column = text_cols[0]
                texts = [item for item in split_data[text_column] if item and str(item).strip()]
            else:
                raise ValueError(f"Text column not found, available columns: {available_cols}")
        
        print(f"Success load {len(texts)} texts")
        return texts
        
    except Exception as e:
        print(f"Dataset Load Error: {e}")
        raise

if __name__ == "__main__":
    current_dir = Path(__file__).parent
    dataset_name = "nikrog/psychology_30_v1"
    model_name = "nikrog/rugpt3small_finetuned_psychology_v2"
    # Загрузка данных и модели
    text_data = load_dataset_from_hf(dataset_name)
    model, tokenizer = load_model_and_tokenizer_from_hf(model_name)
    
    # Вычисление перплексии
    perplexity = calculate_perplexity(model, tokenizer, text_data)
    
    # Сохранение результатов
    print(f"Perplexity: {perplexity:.4f}")
    
    results_path = current_dir / "perplexity_results.json"

    results = {
            'Model': model_name,
            'Dataset': dataset_name,
            'File': "testv2",
            'perplexity': float(perplexity),
            'Test samples': {len(text_data)},
            'timestamp': np.datetime64('now').astype(str)
        }
        
    with open(results_path, 'w', encoding='utf-8') as f:
        json.dump(results, f, indent=2, ensure_ascii=False)
    
    print(f"Результаты сохранены в: {results_path}")
