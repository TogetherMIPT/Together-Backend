import torch
import os
import json
from transformers import AutoModelForCausalLM, AutoTokenizer
from huggingface_hub import login

class PsychologyModel:
    def __init__(self, model_name: str, hf_token: str):
        self.model_name = model_name
        self.hf_token = hf_token
        self.model = None
        self.tokenizer = None
        self.device = "cuda" if torch.cuda.is_available() else "cpu"
        print(f"Используемое устройство: {self.device}")
        
        self._load()
    
    def _load(self):
        try:
            print(f"Аутентификация в Hugging Face Hub...")
            login(token=self.hf_token)
            print("Успешная аутентификация")
            
            print(f"Загрузка модели {self.model_name}...")
            self.model = AutoModelForCausalLM.from_pretrained(
                self.model_name,
                trust_remote_code=True,
                token=self.hf_token,
                device_map="auto" if self.device == "cuda" else None,
                torch_dtype=torch.float16 if self.device == "cuda" else torch.float32
            )
            
            print(f"Загрузка токенизатора...")
            self.tokenizer = AutoTokenizer.from_pretrained(
                self.model_name,
                trust_remote_code=True,
                token=self.hf_token
            )
            
            if self.tokenizer.pad_token is None:
                self.tokenizer.pad_token = self.tokenizer.eos_token
            
            self.model.eval()
            print(f"Модель успешно загружена на {self.device}")
            
        except Exception as e:
            print(f"Ошибка загрузки модели: {e}")
            raise
    
    def generate_response(self, prompt: str, max_length: int = 200, temperature: float = 0.7) -> str:
        try:
            inputs = self.tokenizer(
                prompt,
                return_tensors="pt",
                padding=True,
                truncation=True,
                max_length=512
            ).to(self.device)
            
            with torch.no_grad():
                outputs = self.model.generate(
                    **inputs,
                    max_new_tokens=max_length,
                    temperature=temperature,
                    do_sample=True,
                    top_p=0.9,
                    top_k=50,
                    pad_token_id=self.tokenizer.eos_token_id,
                    eos_token_id=self.tokenizer.eos_token_id
                )
            
            response = self.tokenizer.decode(outputs[0], skip_special_tokens=True)
            
            # Убираем промпт из ответа
            if response.startswith(prompt):
                response = response[len(prompt):].strip()
            
            return response.strip()
        
        except Exception as e:
            print(f"Ошибка генерации: {e}")
            return "Извините, произошла ошибка при генерации ответа."

# Глобальный экземпляр (будет создан при запуске сервиса)
_model = None

def get_model() -> PsychologyModel:
    global _model
    if _model is None:
        hf_token = os.getenv("HF_TOKEN")
        if not hf_token:
            raise ValueError("HF_TOKEN не установлен в переменных окружения")
        
        model_name = os.getenv("MODEL_NAME", "nikrog/rugpt3small_finetuned_psychology_v2")
        _model = PsychologyModel(model_name, hf_token)
    
    return _model