FROM python:3.11-slim
WORKDIR /app
COPY pyproject.toml ./
COPY codelens_sidecar ./codelens_sidecar
RUN pip install --no-cache-dir .
EXPOSE 8000
CMD ["uvicorn", "codelens_sidecar.app:app", "--host", "0.0.0.0", "--port", "8000"]
