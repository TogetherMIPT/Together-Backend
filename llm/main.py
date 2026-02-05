import os
import time
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from typing import Optional
from model_loader import get_model

app = FastAPI(title="Psychology LLM API", version="1.0.0")

# CORS для разработки
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

class GenerateRequest(BaseModel):
    prompt: str
    max_length: Optional[int] = 200
    temperature: Optional[float] = 0.7

class GenerateResponse(BaseModel):
    response: str
    model: str
    processing_time_ms: int

@app.on_event("startup")
async def startup_event():
    print("🧠 Загрузка модели при старте сервиса...")
    start = time.time()
    model = get_model()
    elapsed = (time.time() - start) * 1000
    print(f"✅ Модель загружена за {elapsed:.0f} мс")
    print(f"📦 Модель: {model.model_name}")
    print(f"⚙️ Устройство: {model.device}")

@app.get("/health")
async def health_check():
    model = get_model()
    return {
        "status": "ok",
        "model": model.model_name,
        "device": str(model.device),
        "ready": True
    }

@app.post("/generate", response_model=GenerateResponse)
async def generate(request: GenerateRequest):
    start = time.time()
    
    try:
        model = get_model()
        response = model.generate_response(
            prompt=request.prompt,
            max_length=request.max_length,
            temperature=request.temperature
        )
        
        processing_time = int((time.time() - start) * 1000)
        
        return GenerateResponse(
            response=response,
            model=model.model_name,
            processing_time_ms=processing_time
        )
    
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Ошибка генерации: {str(e)}")

@app.get("/")
async def root():
    return {
        "service": "Psychology LLM API",
        "model": os.getenv("MODEL_NAME", "nikrog/rugpt3small_finetuned_psychology_v2"),
        "endpoints": {
            "health": "GET /health",
            "generate": "POST /generate"
        }
    }

if __name__ == "__main__":
    import uvicorn
    port = int(os.getenv("PORT", 8000))
    uvicorn.run(app, host="0.0.0.0", port=port, log_level="info")