# services/ml-svc/Dockerfile
# Mock mode by default: no weights, no torch, starts in about a second.
# Build with --build-arg EXTRAS=real for the image that actually runs models.
FROM python:3.11-slim
ARG EXTRAS=

COPY --from=ghcr.io/astral-sh/uv:0.4.9 /uv /bin/uv

WORKDIR /srv
ENV UV_SYSTEM_PYTHON=1 UV_LINK_MODE=copy PYTHONUNBUFFERED=1 PYTHONDONTWRITEBYTECODE=1

COPY pyproject.toml ./
COPY app ./app
COPY pb ./pb

RUN uv pip install --system ".${EXTRAS:+[$EXTRAS]}"

EXPOSE 50055 9095
CMD ["python", "-m", "app.server"]
