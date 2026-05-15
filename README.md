# Invest Web System

Web-система анализа рынка ценных бумаг с использованием машинного обучения.

Система позволяет отслеживать инструменты, получать рыночные данные в реальном времени (через Tinkoff Invest API), генерировать ML-сигналы и настраивать уведомления.

## Основные возможности

- **Авторизация**: Регистрация и вход пользователей, JWT-аутентификация.
- **Интеграция с Tinkoff**: Подключение персонального токена Tinkoff Invest API (AES-GCM шифрование).
- **Watchlist**: Персональный список отслеживаемых инструментов.
- **Поиск инструментов**: Поиск по тикеру или названию через Tinkoff API.
- **Детальная страница инструмента**:
  - Интерактивный график (Candlestick Chart) с использованием Lightweight Charts.
  - Технические индикаторы (EMA, RSI, ATR).
  - История сигналов.
  - Ручной запуск анализа и обновления данных.
- **Уведомления**:
  - Настройка правил уведомлений (по тикеру, типу события, порогу вероятности).
  - История сработавших алертов.
- **Admin ML Dashboard**:
  - Мониторинг состояния системы.
  - Управление активными моделями.
  - Просмотр результатов исследований и валидации.
  - История фоновых задач (Data Refresh, Signal Run).

## Технологический стек

### Backend
- **Язык**: Go 1.22+
- **Framework**: Standard Library (`net/http`) with custom handlers.
- **Database**: PostgreSQL.
- **Auth**: JWT (Access & Refresh tokens).
- **Security**: AES-GCM для хранения API ключей пользователей.

### Frontend
- **Framework**: React 18 (TypeScript).
- **Routing**: React Router 7.
- **State Management**: React Hooks.
- **Charts**: Lightweight Charts.
- **Styling**: Vanilla CSS.

### ML Core (Python)
- **Frameworks**: PyTorch, Scikit-learn, Pandas.
- **Models**: GRU (Neural Sequence Model), Gradient Boosting (Baseline).
- **Research**: Walk-forward validation, Calibration audit.

## Быстрый старт

### Требования
- Docker & Docker Compose
- Go 1.22+
- Node.js 20+
- Python 3.10+ (для ML Core)

### Установка и запуск

1. **Клонирование и настройка**:
   ```sh
   cp .env.example .env
   ```

2. **Запуск базы данных**:
   ```sh
   docker compose -f infra/compose.yaml up -d postgres
   ```

3. **Запуск Backend**:
   ```sh
   make backend-migrate
   make backend-run
   ```

4. **Запуск Frontend**:
   ```sh
   cd frontend
   npm install
   npm run dev
   ```

5. **Настройка ML Core (опционально)**:
   ```sh
   python -m venv .venv
   source .venv/bin/activate
   make ml-install
   ```

## Документация

- [План разработки (V2)](DEVELOPMENT_PLAN_V2.md)
- [Индекс документации](docs/README.md)
- [Архитектура системы](docs/architecture/overview.md)
- [API Контракт (V1)](docs/contracts/api_v1.md)
- [ML Objective](docs/ml/ml_objective_v1.md)
- [Research Protocol](docs/ml/research_protocol_v1.md)

## Автор
ВКР: Веб-система анализа рынка ценных бумаг.
