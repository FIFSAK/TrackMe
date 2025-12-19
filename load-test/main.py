"""
TrackMe API Load Test
Запуск: locust -f main.py --host=http://localhost:80
"""

from locust import HttpUser, task, between
from random import randint, choice
import uuid
import logging
from datetime import datetime

# Настройка логирования
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

VALID_STAGES = [
    "registration",
    "product_selection",
    "data_consent",
    "form_filling",
    "participants_specification",
    "terms_agreement",
    "client_questionnaire",
    "approval_waiting",
    "modifications",
    "document_signing",
    "payment_waiting",
    "completed"
]


def to_rfc3339(dt):
    if isinstance(dt, str):
        return dt
    if isinstance(dt, datetime):
        return dt.strftime("%Y-%m-%dT%H:%M:%SZ")
    return "2024-12-15T10:30:00Z"  # fallback


class TrackMeUser(HttpUser):
    """
    Основной класс пользователя для нагрузочного тестирования
    """
    wait_time = between(1, 3)  # Задержка между запросами 1-3 секунды

    def on_start(self):
        """Инициализация и авторизация пользователя"""
        self.token = None
        self.user_id = None
        self.client_ids = []
        self.email = f"user_{uuid.uuid4().hex[:8]}@test.com"
        self.password = "TestPass123!"
        self.register_and_login()

    def register_and_login(self):
        """Регистрация нового пользователя и получение токена"""
        # Пытаемся зарегистрироваться
        register_payload = {
            "email": self.email,
            "name": f"Test User {randint(1000, 9999)}",
            "password": self.password,
            "role": "user"
        }

        with self.client.post(
                "/api/v1/auth/register",
                json=register_payload,
                catch_response=True,
                name="/auth/register"
        ) as response:
            if response.status_code == 201:
                try:
                    data = response.json()
                    # API возвращает данные в обёртке {"data": {...}}
                    response_data = data.get("data", {})
                    self.token = response_data.get("token")
                    self.user_id = response_data.get("user", {}).get("id")
                    if self.token:
                        response.success()
                    else:
                        response.failure(f"No token in response: {data}")
                except Exception as e:
                    response.failure(f"Failed to parse response: {e}")
            elif response.status_code == 409:
                # Пользователь уже существует, пробуем залогиниться
                response.success()
                self.login(self.email, self.password)
            elif response.status_code == 0:
                response.failure("Connection refused - server not running?")
            else:
                response.failure(f"Registration failed: {response.status_code} - {response.text[:200]}")

    def login(self, email, password):
        """Логин существующего пользователя"""
        login_payload = {
            "email": email,
            "password": password
        }

        with self.client.post(
                "/api/v1/auth/login",
                json=login_payload,
                catch_response=True,
                name="/auth/login"
        ) as response:
            if response.status_code == 200:
                try:
                    data = response.json()
                    # API возвращает данные в обёртке {"data": {...}}
                    response_data = data.get("data", {})
                    self.token = response_data.get("token")
                    self.user_id = response_data.get("user", {}).get("id")
                    if self.token:
                        response.success()
                    else:
                        response.failure(f"No token in login response: {data}")
                except Exception as e:
                    response.failure(f"Failed to parse login response: {e}")
            elif response.status_code == 0:
                response.failure("Connection refused - server not running?")
            else:
                response.failure(f"Login failed: {response.status_code} - {response.text[:200]}")

    def get_auth_headers(self):
        """Получение заголовков с токеном авторизации"""
        if self.token:
            return {"Authorization": f"Bearer {self.token}"}
        return {}

    @task(10)
    def list_clients(self):
        """Получение списка клиентов с различными фильтрами"""
        if not self.token:
            logger.warning("list_clients skipped - no token available")
            return

        filters = [
            {},
            {"is_active": "true", "limit": 50},
            {"stage": "active", "limit": 20},
            {"source": "web", "channel": "organic"},
            {"app": "installed", "limit": 30, "offset": 10},
        ]

        params = choice(filters)

        with self.client.get(
                "/api/v1/clients",
                params=params,
                headers=self.get_auth_headers(),
                catch_response=True,
                name="/clients [LIST]"
        ) as response:
            if response.status_code == 200:
                response.success()
            elif response.status_code == 401:
                response.failure("Unauthorized - token may be invalid")
                self.token = None  # Сбрасываем токен для повторной авторизации
            else:
                response.failure(f"Failed to list clients: {response.status_code} - {response.text[:100]}")

    @task(6)
    def create_client(self):
        """Создание нового клиента"""
        if not self.token:
            logger.warning("create_client skipped - no token available")
            return

        client_data = {
            "name": f"Client {uuid.uuid4().hex[:8]}",
            "email": f"client_{uuid.uuid4().hex[:8]}@example.com",
            "stage": choice(VALID_STAGES),
            "source": choice(["web", "mobile", "referral", "ads"]),
            "channel": choice(["organic", "paid", "direct"]),
            "app": choice(["not_installed", "installed", "active"]),
            "is_active": True,
            "last_login": "2024-12-15T10:30:00Z",
            "contracts": [
                {
                    "name": f"Contract {randint(1000, 9999)}",
                    "number": f"CNT-{randint(10000, 99999)}",
                    "amount": round(randint(1000, 50000) + 0.99, 2),
                    "status": choice(["active", "pending", "expired"]),
                    "payment_frequency": choice(["monthly", "quarterly", "annual"]),
                    "autopayment": choice(["enabled", "disabled", "pending"]),
                    "conclusion_date": "2024-01-01T00:00:00Z",
                    "expiration_date": "2025-01-01T00:00:00Z"
                }
            ]
        }

        with self.client.post(
                "/api/v1/clients",
                json=client_data,
                headers=self.get_auth_headers(),
                catch_response=True,
                name="/clients [CREATE]"
        ) as response:
            if response.status_code == 201:
                try:
                    data = response.json()
                    response_data = data.get("data", data)
                    # Сохраняем все данные клиента для последующих операций
                    self.client_ids.append(response_data)
                    response.success()
                except Exception as e:
                    response.failure(f"Failed to parse create response: {e}")
            elif response.status_code == 401:
                response.failure("Unauthorized - token may be invalid")
                self.token = None
            else:
                response.failure(f"Failed to create client: {response.status_code} - {response.text[:100]}")

    @task(4)
    def update_client_stage(self):
        """Обновление стадии клиента"""
        if not self.token:
            logger.warning("update_client_stage skipped - no token available")
            return
        if not self.client_ids:
            logger.debug("update_client_stage skipped - no client_ids available")
            return

        client_data = choice(self.client_ids)
        client_id = client_data.get("id")
        client_data["stage"] = choice(VALID_STAGES)
        client_data["is_active"] = choice([True, False])

        # Преобразуем last_login
        if "last_login" in client_data:
            client_data["last_login"] = to_rfc3339(client_data["last_login"])

        # Преобразуем даты и autopayment в каждом контракте
        if "contracts" in client_data:
            for contract in client_data["contracts"]:
                if "conclusion_date" in contract:
                    contract["conclusion_date"] = to_rfc3339(contract["conclusion_date"])
                if "expiration_date" in contract:
                    contract["expiration_date"] = to_rfc3339(contract["expiration_date"])
                # Преобразуем autopayment в строку, если это не строка
                if "autopayment" in contract and not isinstance(contract["autopayment"], str):
                    ap_val = contract["autopayment"]
                    if isinstance(ap_val, dict):
                        contract["autopayment"] = ap_val.get("status") or ap_val.get("name") or str(ap_val)
                    else:
                        contract["autopayment"] = str(ap_val)

        # Преобразуем app в строку, если это объект
        if "app" in client_data and not isinstance(client_data["app"], str):
            app_val = client_data["app"]
            if isinstance(app_val, dict):
                client_data["app"] = app_val.get("status") or app_val.get("name") or str(app_val)
            else:
                client_data["app"] = str(app_val)

        with self.client.put(
            f"/api/v1/clients/{client_id}/stage",
            json=client_data,
            headers=self.get_auth_headers(),
            catch_response=True,
            name="/clients/{id}/stage [UPDATE]"
        ) as response:
            if response.status_code in [200, 201]:
                response.success()
            elif response.status_code == 404:
                self.client_ids = [c for c in self.client_ids if c.get("id") != client_id]
                response.failure(f"Client not found: {client_id}")
            else:
                response.failure(f"Failed to update client: {response.status_code} - {response.text[:100]}")

    @task(8)
    def get_metrics(self):
        """Получение метрик с различными фильтрами"""
        if not self.token:
            logger.warning("get_metrics skipped - no token available")
            return

        filters = [
            {},
            {"type": "clients", "interval": "day"},
            {"type": "revenue", "interval": "week"},
            {"type": "active_users", "interval": "month"},
            {"interval": "day"},
        ]

        params = choice(filters)

        with self.client.get(
                "/api/v1/metrics",
                params=params,
                headers=self.get_auth_headers(),
                catch_response=True,
                name="/metrics [GET]"
        ) as response:
            if response.status_code == 200:
                response.success()
            elif response.status_code == 401:
                response.failure("Unauthorized")
                self.token = None
            else:
                response.failure(f"Failed to get metrics: {response.status_code} - {response.text[:100]}")

    @task(5)
    def list_users(self):
        """Получение списка пользователей"""
        if not self.token:
            logger.warning("list_users skipped - no token available")
            return

        params = {
            "limit": randint(10, 50),
            "offset": randint(0, 20)
        }

        with self.client.get(
                "/api/v1/users",
                params=params,
                headers=self.get_auth_headers(),
                catch_response=True,
                name="/users [LIST]"
        ) as response:
            if response.status_code == 200:
                response.success()
            elif response.status_code == 401:
                response.failure("Unauthorized")
                self.token = None
            else:
                response.failure(f"Failed to list users: {response.status_code} - {response.text[:100]}")

    @task(2)
    def get_current_user(self):
        """Получение информации о текущем пользователе"""
        if not self.token or not self.user_id:
            logger.warning("get_current_user skipped - no token or user_id available")
            return

        with self.client.get(
                f"/api/v1/users/{self.user_id}",
                headers=self.get_auth_headers(),
                catch_response=True,
                name="/users/{id} [GET]"
        ) as response:
            if response.status_code == 200:
                response.success()
            elif response.status_code == 401:
                response.failure("Unauthorized")
                self.token = None
            else:
                response.failure(f"Failed to get user: {response.status_code} - {response.text[:100]}")

    @task(1)
    def delete_client(self):
        """Удаление клиента"""
        if not self.token:
            logger.warning("delete_client skipped - no token available")
            return
        if len(self.client_ids) < 2:
            logger.debug("delete_client skipped - not enough client_ids")
            return

        client_data = self.client_ids.pop()
        client_id = client_data.get("id")

        with self.client.delete(
                f"/api/v1/clients/{client_id}",
                headers=self.get_auth_headers(),
                catch_response=True,
                name="/clients/{id} [DELETE]"
        ) as response:
            if response.status_code == 204:
                response.success()
            elif response.status_code == 404:
                response.failure(f"Client not found: {client_id}")
            else:
                response.failure(f"Failed to delete client: {response.status_code} - {response.text[:100]}")


class AdminUser(HttpUser):
    """
    Продвинутый пользователь с повышенной активностью
    """
    wait_time = between(0.5, 2)  # Более частые запросы
    weight = 2  # Меньший вес по сравнению с обычными пользователями

    def on_start(self):
        """Инициализация администратора"""
        self.token = None
        self.user_id = None
        self.client_ids = []
        self.register_and_login()

    def register_and_login(self):
        """Регистрация и логин администратора"""
        self.email = f"admin_{uuid.uuid4().hex[:8]}@test.com"
        self.password = "AdminPass123!"

        register_payload = {
            "email": self.email,
            "name": f"Admin User {randint(1000, 9999)}",
            "password": self.password,
            "role": "admin"
        }

        response = self.client.post(
            "/api/v1/auth/register",
            json=register_payload,
            name="/auth/register [ADMIN]"
        )

        if response.status_code == 201:
            data = response.json()
            response_data = data.get("data", {})
            self.token = response_data.get("token")
            self.user_id = response_data.get("user", {}).get("id")
        elif response.status_code == 409:
            # Пользователь уже существует, пробуем залогиниться
            self.login_admin()

    def login_admin(self):
        """Логин администратора"""
        login_payload = {"email": self.email, "password": self.password}
        response = self.client.post(
            "/api/v1/auth/login",
            json=login_payload,
            name="/auth/login [ADMIN]"
        )

        if response.status_code == 200:
            data = response.json()
            response_data = data.get("data", {})
            self.token = response_data.get("token")
            self.user_id = response_data.get("user", {}).get("id")

    def get_auth_headers(self):
        """Получение заголовков с токеном авторизации"""
        if self.token:
            return {"Authorization": f"Bearer {self.token}"}
        return {}

    @task(15)
    def intensive_client_list(self):
        """Интенсивное получение списка клиентов"""
        if not self.token:
            return

        self.client.get(
            "/api/v1/clients",
            params={"limit": 100, "is_active": "true"},
            headers=self.get_auth_headers(),
            name="/clients [ADMIN LIST]"
        )

    @task(8)
    def create_multiple_clients(self):
        """Создание клиента администратором"""
        if not self.token:
            return

        client_data = {
            "name": f"Admin Client {uuid.uuid4().hex[:6]}",
            "email": f"admin_client_{uuid.uuid4().hex[:8]}@example.com",
            "stage": choice(VALID_STAGES),
            "source": "web",
            "channel": "direct",
            "app": "installed",
            "is_active": True,
            "last_login": "2024-12-15T10:30:00Z",
            "contracts": []
        }

        response = self.client.post(
            "/api/v1/clients",
            json=client_data,
            headers=self.get_auth_headers(),
            name="/clients [ADMIN CREATE]"
        )

        if response.status_code == 201:
            data = response.json()
            response_data = data.get("data", data)
            client_id = response_data.get("id")
            if client_id:
                self.client_ids.append(client_id)

    @task(10)
    def check_metrics(self):
        """Проверка метрик администратором"""
        if not self.token:
            return

        self.client.get(
            "/api/v1/metrics",
            params={"interval": "day"},
            headers=self.get_auth_headers(),
            name="/metrics [ADMIN]"
        )

    @task(5)
    def manage_users(self):
        """Управление пользователями"""
        if not self.token:
            return

        self.client.get(
            "/api/v1/users",
            params={"limit": 100},
            headers=self.get_auth_headers(),
            name="/users [ADMIN LIST]"
        )