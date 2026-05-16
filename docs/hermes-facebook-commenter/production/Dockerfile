FROM python:3.12-slim

WORKDIR /opt/hermes-webhook

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

RUN mkdir -p /var/log/hermes /root/.hermes/webhook_queue

EXPOSE 5000

CMD ["gunicorn", "--bind", "0.0.0.0:5000", "--workers", "4", "--timeout", "120", "facebook_webhook_gunicorn:app"]
