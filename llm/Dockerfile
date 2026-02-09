# STAGE 1: Базовый образ с CUDA
FROM nvidia/cuda:12.1.1-cudnn8-runtime-ubuntu22.04 AS base

RUN apt-get update && apt-get install -y --no-install-recommends \
    python3.10 python3-pip python3-venv git curl \
    && rm -rf /var/lib/apt/lists/*

# STAGE 2: Установка зависимостей в отдельном слое
FROM base AS dependencies

WORKDIR /app
COPY requirements.txt .
# Объединяем установку и очистку в один слой для экономии места
RUN pip3 install --no-cache-dir -r requirements.txt \
    && find /usr/local/lib/python3.10/dist-packages -name "*.so" -type f -exec strip --strip-unneeded {} \; 2>/dev/null || true \
    && find /usr/local/lib/python3.10/dist-packages -name "*.pyc" -delete \
    && find /usr/local/lib/python3.10/dist-packages -name "__pycache__" -delete

# STAGE 3: Минимальный финальный образ
FROM base

WORKDIR /app

# Копируем ТОЛЬКО установленные пакеты и бинарники
COPY --from=dependencies /usr/local/lib/python3.10/dist-packages /usr/local/lib/python3.10/dist-packages
COPY --from=dependencies /usr/local/bin /usr/local/bin

# Копируем только необходимые файлы приложения (исключаем .git, __pycache__, etc.)
COPY --chown=1000:1000 . .

# Создаём непривилегированного пользователя
RUN useradd -m -u 1000 appuser && chown -R appuser:appuser /app
USER appuser

EXPOSE 8000
CMD ["python3", "main.py"]