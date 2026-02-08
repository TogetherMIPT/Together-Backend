# STAGE 1: Сборка с кэшированием зависимостей
FROM nvidia/cuda:12.1.1-cudnn8-runtime-ubuntu22.04 AS base

# Установка Python и зависимостей
RUN apt-get update && apt-get install -y \
    python3.10 \
    python3-pip \
    python3-venv \
    git \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Установка torch с поддержкой CUDA
#RUN pip3 install --no-cache-dir torch==2.1.0+cu121 torchvision==0.16.0+cu121 torchaudio==2.1.0 --index-url https://download.pytorch.org/whl/cu121

# STAGE 2: Установка остальных зависимостей
#FROM base AS dependencies

WORKDIR /app

COPY requirements.txt .
RUN pip3 install --no-cache-dir -r requirements.txt

# STAGE 3: Финальный образ
#FROM base

#WORKDIR /app

# Копируем зависимости из предыдущего слоя
#COPY --from=dependencies /usr/local/lib/python3.10/dist-packages /usr/local/lib/python3.10/dist-packages
#COPY --from=dependencies /usr/local/bin /usr/local/bin

# Копируем код приложения
COPY . .

# Создаём пользователя без прав root
RUN useradd -m -u 1000 appuser && chown -R appuser:appuser /app
USER appuser

EXPOSE 8000

# Healthcheck
#HEALTHCHECK --interval=30s --timeout=10s --start-period=300s --retries=3 \
#  CMD curl -f http://localhost:8000/health || exit 1

CMD ["python3", "main.py"]
