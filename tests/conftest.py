# =============================================================================
# Client Hub Open Project
# Copyright (C) 2025 Client Hub Contributors
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.
# =============================================================================

import pytest
import requests
import time
import os
import itertools
from dotenv import load_dotenv

# Load environment variables from .env file securely before any tests run
load_dotenv(override=True)



def _unlock_root_via_db():
    """
    Reset root user's lock state directly in PostgreSQL.

    This prevents cascading test failures when root gets locked by
    test_login_blocking.py (or any test that intentionally triggers
    failed login attempts on root). Without this, a locked root from
    a previous test run causes ~90% of the suite to fail because
    every fixture that depends on root_token gets None.
    """
    try:
        import psycopg2
    except ImportError:
        print("⚠️  psycopg2 not installed — cannot reset root lock via DB")
        return False

    db_host = os.getenv("DB_HOST", os.getenv("POSTGRES_HOST", "localhost"))
    db_port = os.getenv("DB_PORT", os.getenv("POSTGRES_PORT", "5432"))

    # Try multiple credential/dbname combinations to cover dev, test, and CI envs.
    # The first successful connection wins.
    candidates = []

    # 1. Explicit env vars (highest priority)
    env_user = os.getenv("DB_USER") or os.getenv("POSTGRES_USER")
    env_pass = os.getenv("DB_PASSWORD") or os.getenv("POSTGRES_PASSWORD")
    env_db = os.getenv("DB_NAME") or os.getenv("POSTGRES_DB")
    if env_user and env_db:
        candidates.append((env_user, env_pass or "", env_db))

    # 2. Common dev environment (chopdb_dev)
    candidates.append(("chopuser_dev", "THIS_IS_A_DEV_ENVIRONMENT_PASSWORD!123abc", "chopdb_dev"))

    # 3. Docker test environment
    candidates.append(("test_user", "test_password", "contracts_test"))

    # 4. Generic postgres defaults
    candidates.append(("postgres", "postgres", "chopdb_dev"))
    candidates.append(("postgres", "postgres", "contracts_manager"))

    for db_user, db_password, db_name in candidates:
        try:
            conn = psycopg2.connect(
                host=db_host,
                port=db_port,
                user=db_user,
                password=db_password,
                dbname=db_name,
                connect_timeout=3,
            )
            conn.autocommit = True
            cur = conn.cursor()
            cur.execute(
                "UPDATE users SET failed_attempts = 0, lock_level = 0, locked_until = NULL "
                "WHERE username = 'root' AND deleted_at IS NULL"
            )
            rows = cur.rowcount
            cur.close()
            conn.close()
            if rows > 0:
                print(f"🔓 Root user lock reset via DB [{db_name}] (was locked from previous run)")
            return True
        except Exception:
            continue

    print("⚠️  Could not reset root lock via DB (all connection candidates failed)")
    return False

# Configuração base - usando portas de teste
# Configuração base - usando portas de teste
raw_url = os.getenv("API_URL") or os.getenv("TEST_API_URL", "http://localhost:3000/api")
BASE_URL = raw_url.rstrip("/")
if BASE_URL.endswith("/api"):
    BASE_URL = BASE_URL[:-4]

API_URL = f"{BASE_URL}/api"

# Armazenamento de tokens e usuários criados
test_data = {
    "tokens": {},
    "users": {},
    "clients": [],
    "contracts": [],
    "categories": [],
    "subcategories": []
}

# Senhas padrão para testes
DEFAULT_ROOT_PASSWORD = os.getenv(
    "TEST_ROOT_PASSWORD",
    "THIS_IS_A_DEV_ENVIRONMENT_PASSWORD!123abc"
)
DEFAULT_STRONG_PASSWORD = "ValidPass123!@#abcXYZ"

_ip_counter = itertools.count(1)


def _next_ip() -> str:
    # Use a deterministic, rotating IP to avoid rate limiting during tests
    octet = next(_ip_counter) % 254 + 1
    return f"10.99.0.{octet}"


@pytest.fixture(scope="session")
def base_url():
    return BASE_URL


@pytest.fixture(scope="session")
def api_url():
    return API_URL


@pytest.fixture(scope="session")
def http_client():
    session = requests.Session()
    session.headers.update({"Content-Type": "application/json"})

    original_request = session.request

    def request_with_ip(method, url, **kwargs):
        headers = kwargs.pop("headers", {}) or {}
        headers.setdefault("X-Forwarded-For", _next_ip())
        kwargs["headers"] = headers
        return original_request(method, url, **kwargs)

    setattr(session, "request", request_with_ip)
    return session


@pytest.fixture(scope="session")
def root_user(http_client, api_url, setup_teardown):
    """Cria ou usa usuário root existente.

    Depends on setup_teardown explicitly so the DB-level unlock runs first.
    If root is still locked (e.g. DB reset failed), attempts a second
    DB-level unlock before giving up.
    """
    # Tentar criar root admin (só funciona se banco vazio)
    data = {
        "username": "root",
        "display_name": "Root Admin",
        "password": DEFAULT_ROOT_PASSWORD
    }

    response = http_client.post(f"{api_url}/initialize/admin", json=data)

    if response.status_code == 200:
        result = response.json()
        test_data["users"]["root"] = {
            "username": result.get("admin_username", "root"),
            "password": data["password"],
            "id": result.get("admin_id"),
            "role": "root"
        }
    else:
        # Root já existe, usar credenciais padrão
        test_data["users"]["root"] = {
            "username": "root",
            "password": DEFAULT_ROOT_PASSWORD,
            "role": "root"
        }

    # Fazer login
    login_response = http_client.post(f"{api_url}/login", json={
        "username": test_data["users"]["root"]["username"],
        "password": test_data["users"]["root"]["password"]
    })

    # If root is locked (423), try DB unlock one more time and retry
    if login_response.status_code == 423:
        print("⚠️  Root is locked (423) — attempting DB-level unlock retry...")
        _unlock_root_via_db()
        time.sleep(0.5)
        login_response = http_client.post(f"{api_url}/login", json={
            "username": test_data["users"]["root"]["username"],
            "password": test_data["users"]["root"]["password"]
        })

    if login_response.status_code == 200:
        tokens = login_response.json()
        # Suporta ambos os formatos de resposta
        access_token = tokens.get("token") or tokens.get("access_token") or tokens.get("data", {}).get("token")
        refresh_token = tokens.get("refresh_token") or tokens.get("data", {}).get("refresh_token")
        test_data["tokens"]["root"] = access_token
        test_data["users"]["root"]["token"] = access_token
        test_data["users"]["root"]["refresh_token"] = refresh_token
        test_data["users"]["root"]["id"] = tokens.get("user_id") or tokens.get("data", {}).get("user_id")
    else:
        # Se login falhou, tenta com senha alternativa
        alt_passwords = [
            "RootPass123!@#",
            DEFAULT_STRONG_PASSWORD,
        ]
        for alt_pass in alt_passwords:
            login_response = http_client.post(f"{api_url}/login", json={
                "username": "root",
                "password": alt_pass
            })
            if login_response.status_code == 200:
                tokens = login_response.json()
                access_token = tokens.get("token") or tokens.get("access_token") or tokens.get("data", {}).get("token")
                refresh_token = tokens.get("refresh_token") or tokens.get("data", {}).get("refresh_token")
                test_data["tokens"]["root"] = access_token
                test_data["users"]["root"]["token"] = access_token
                test_data["users"]["root"]["refresh_token"] = refresh_token
                test_data["users"]["root"]["password"] = alt_pass
                break

    if "token" not in test_data["users"]["root"]:
        print(f"❌ CRITICAL: Could not obtain root token (last status: {login_response.status_code})")
        print(f"   Most tests will be SKIPPED. Check root credentials or DB connectivity.")

    return test_data["users"]["root"]


@pytest.fixture(scope="session")
def admin_user(http_client, api_url, root_user):
    """Cria usuário admin. Skips if root token is unavailable."""
    if not root_user or "token" not in root_user:
        pytest.skip("admin_user requires root token (root login failed)")

    headers = {"Authorization": f"Bearer {root_user['token']}"}

    admin_username = f"admin_test_{int(time.time())}"
    data = {
        "username": admin_username,
        "display_name": "Admin Test User",
        "password": DEFAULT_STRONG_PASSWORD,
        "role": "admin"
    }

    response = http_client.post(f"{api_url}/users", json=data, headers=headers)

    if response.status_code in [200, 201]:
        result = response.json()
        user_data = {
            "username": data["username"],
            "password": data["password"],
            "id": result.get("id") or result.get("data", {}).get("id"),
            "role": "admin"
        }

        # Fazer login
        login_response = http_client.post(f"{api_url}/login", json={
            "username": user_data["username"],
            "password": user_data["password"]
        })

        if login_response.status_code == 200:
            tokens = login_response.json()
            access_token = tokens.get("token") or tokens.get("access_token") or tokens.get("data", {}).get("token")
            refresh_token = tokens.get("refresh_token") or tokens.get("data", {}).get("refresh_token")
            user_data["token"] = access_token
            user_data["refresh_token"] = refresh_token
            test_data["tokens"]["admin"] = access_token
            test_data["users"]["admin"] = user_data

        return user_data

    return None


@pytest.fixture(scope="session")
def regular_user(http_client, api_url, root_user):
    """Cria usuário comum. Skips if root token is unavailable."""
    if not root_user or "token" not in root_user:
        pytest.skip("regular_user requires root token (root login failed)")

    headers = {"Authorization": f"Bearer {root_user['token']}"}

    user_username = f"user_test_{int(time.time())}"
    data = {
        "username": user_username,
        "display_name": "Regular Test User",
        "password": DEFAULT_STRONG_PASSWORD,
        "role": "user"
    }

    response = http_client.post(f"{api_url}/users", json=data, headers=headers)

    if response.status_code in [200, 201]:
        result = response.json()
        user_data = {
            "username": data["username"],
            "password": data["password"],
            "id": result.get("id") or result.get("data", {}).get("id"),
            "role": "user"
        }

        # Fazer login
        login_response = http_client.post(f"{api_url}/login", json={
            "username": user_data["username"],
            "password": user_data["password"]
        })

        if login_response.status_code == 200:
            tokens = login_response.json()
            access_token = tokens.get("token") or tokens.get("access_token") or tokens.get("data", {}).get("token")
            refresh_token = tokens.get("refresh_token") or tokens.get("data", {}).get("refresh_token")
            user_data["token"] = access_token
            user_data["refresh_token"] = refresh_token
            test_data["tokens"]["user"] = access_token
            test_data["users"]["user"] = user_data

        return user_data

    return None


@pytest.fixture
def timer():
    """Fixture para medir tempo de testes"""
    start = time.time()
    yield
    duration = time.time() - start
    print(f"\n⏱️  Test duration: {duration:.3f}s")


@pytest.fixture(scope="function")
def test_timing(request):
    """Hook para registrar timing de cada teste"""
    start = time.time()
    yield
    duration = time.time() - start
    # Armazena timing no item do teste para relatórios
    request.node.user_properties.append(("duration", duration))


def pytest_configure(config):
    """Configuração inicial do pytest"""
    config.addinivalue_line(
        "markers", "security: marca testes de segurança"
    )
    config.addinivalue_line(
        "markers", "jwt: marca testes de JWT"
    )
    config.addinivalue_line(
        "markers", "sql_injection: marca testes de SQL injection"
    )
    config.addinivalue_line(
        "markers", "xss: marca testes de XSS"
    )
    config.addinivalue_line(
        "markers", "authorization: marca testes de autorização"
    )
    config.addinivalue_line(
        "markers", "api: marca testes de API"
    )
    config.addinivalue_line(
        "markers", "slow: marca testes lentos"
    )
    config.addinivalue_line(
        "markers", "password: marca testes de validação de senha"
    )
    config.addinivalue_line(
        "markers", "validation: marca testes de validação de entrada"
    )
    config.addinivalue_line(
        "markers", "login_blocking: marca testes de bloqueio de login"
    )
    config.addinivalue_line(
        "markers", "rate_limiting: marca testes de rate limiting"
    )
    config.addinivalue_line(
        "markers", "data_leakage: marca testes de vazamento de dados"
    )
    config.addinivalue_line(
        "markers", "initialization: marca testes de inicialização do sistema"
    )
    config.addinivalue_line(
        "markers", "cors: marca testes de CORS security"
    )
    config.addinivalue_line(
        "markers", "headers: marca testes de HTTP security headers"
    )
    config.addinivalue_line(
        "markers", "concurrency: marca testes de concorrência/race conditions"
    )
    config.addinivalue_line(
        "markers", "database: marca testes de resiliência de banco de dados"
    )

    config.addinivalue_line(
        "markers", "compliance: marca testes de conformidade AGPL"
    )
    config.addinivalue_line(
        "markers", "input_validation: marca testes de validação de entrada"
    )


def pytest_collection_modifyitems(config, items):
    """Adiciona marcadores automaticamente baseado no nome do arquivo/teste"""
    _ = config
    for item in items:
        # Marcadores baseados no nome do arquivo
        if "test_jwt" in item.nodeid:
            item.add_marker(pytest.mark.jwt)
            item.add_marker(pytest.mark.security)
        if "test_sql" in item.nodeid:
            item.add_marker(pytest.mark.sql_injection)
            item.add_marker(pytest.mark.security)
        if "test_xss" in item.nodeid:
            item.add_marker(pytest.mark.xss)
            item.add_marker(pytest.mark.security)
        if "test_auth" in item.nodeid or "authorization" in item.nodeid:
            item.add_marker(pytest.mark.authorization)
            item.add_marker(pytest.mark.security)
        if "test_password" in item.nodeid:
            item.add_marker(pytest.mark.password)
            item.add_marker(pytest.mark.security)
        if "test_input" in item.nodeid or "test_validation" in item.nodeid:
            item.add_marker(pytest.mark.validation)
        if "test_login_blocking" in item.nodeid:
            item.add_marker(pytest.mark.login_blocking)
            item.add_marker(pytest.mark.security)
        if "test_data_leakage" in item.nodeid:
            item.add_marker(pytest.mark.data_leakage)
            item.add_marker(pytest.mark.security)
        if "test_api" in item.nodeid:
            item.add_marker(pytest.mark.api)
        if "test_initialization" in item.nodeid:
            item.add_marker(pytest.mark.initialization)
            item.add_marker(pytest.mark.security)
        if "cors" in item.nodeid.lower():
            item.add_marker(pytest.mark.cors)
            item.add_marker(pytest.mark.security)
        if "headers" in item.nodeid.lower():
            item.add_marker(pytest.mark.headers)
            item.add_marker(pytest.mark.security)
        if "concurrency" in item.nodeid.lower():
            item.add_marker(pytest.mark.concurrency)
        if "rate_limiting" in item.nodeid.lower():
            item.add_marker(pytest.mark.rate_limiting)
            item.add_marker(pytest.mark.security)


def pytest_terminal_summary(terminalreporter, exitstatus, config):
    """Adiciona sumário customizado ao final dos testes"""
    _ = exitstatus
    _ = config
    terminalreporter.write_sep("=", "Test Timing Summary")

    # Coletar timings
    timings = []
    for report in terminalreporter.stats.get("passed", []):
        for prop in report.user_properties:
            if prop[0] == "duration":
                timings.append((report.nodeid, prop[1]))

    for report in terminalreporter.stats.get("failed", []):
        for prop in report.user_properties:
            if prop[0] == "duration":
                timings.append((report.nodeid, prop[1]))

    if timings:
        # Ordenar por duração (mais lentos primeiro)
        timings.sort(key=lambda x: x[1], reverse=True)

        terminalreporter.write_line("\nTop 10 slowest tests:")
        for nodeid, duration in timings[:10]:
            terminalreporter.write_line(f"  {duration:.3f}s - {nodeid}")

        total = sum(t[1] for t in timings)
        terminalreporter.write_line(f"\nTotal measured time: {total:.2f}s")


@pytest.fixture(scope="session", autouse=True)
def setup_teardown(http_client, api_url):
    """Setup e teardown da suite de testes"""
    print(f"\n{'='*70}")
    print(f"🚀 INICIANDO SUITE DE TESTES")
    print(f"{'='*70}")
    print(f"📍 Backend URL: {BASE_URL}")
    print(f"📍 API URL: {api_url}")
    print(f"🔧 Test Environment:")
    print(f"   - Base URL: {BASE_URL}")
    print(f"   - API URL: {api_url}")
    print(f"{'='*70}\n")

    # Verificar se backend está online
    try:
        health_response = http_client.get(f"{BASE_URL}/health", timeout=5)
        if health_response.status_code == 200:
            print(f"✅ Backend está online")
        else:
            print(f"⚠️  Backend respondeu com status {health_response.status_code}")
    except Exception as e:
        print(f"❌ Backend não está acessível: {e}")
        print(f"   Certifique-se de que o ambiente de teste está rodando:")
        print(f"   cd tests && docker-compose -f docker-compose.test.yml up -d")

    # Reset root lock state BEFORE any test tries to authenticate.
    # This prevents cascade failures when root was locked by a previous
    # test_login_blocking run that didn't clean up (or was interrupted).
    _unlock_root_via_db()

    yield

    # Also reset root lock at teardown so the dev environment is left clean
    # after tests that intentionally lock accounts (test_login_blocking, etc.)
    _unlock_root_via_db()

    print(f"\n{'='*70}")
    print(f"🧹 FINALIZANDO SUITE DE TESTES")
    print(f"{'='*70}\n")


# Fixtures de tokens para testes de segurança
# These skip instead of returning None so that tests depending on them
# are reported as SKIPPED rather than failing with cryptic 401 errors
# caused by "Bearer None" headers.
@pytest.fixture(scope="session")
def root_token(root_user):
    """Token do usuário root. Skips test if unavailable."""
    if root_user and "token" in root_user:
        return root_user["token"]
    pytest.skip("root_token unavailable (root login failed)")


@pytest.fixture(scope="session")
def admin_token(admin_user):
    """Token do usuário admin. Skips test if unavailable."""
    if admin_user and "token" in admin_user:
        return admin_user["token"]
    pytest.skip("admin_token unavailable (admin user creation failed)")


@pytest.fixture(scope="session")
def user_token(regular_user):
    """Token do usuário comum. Skips test if unavailable."""
    if regular_user and "token" in regular_user:
        return regular_user["token"]
    pytest.skip("user_token unavailable (regular user creation failed)")


# Funções utilitárias para os testes
def get_valid_token(http_client, api_url, username, password):
    """Obtém um token válido para um usuário"""
    response = http_client.post(f"{api_url}/login", json={
        "username": username,
        "password": password
    })
    if response.status_code == 200:
        data = response.json()
        return data.get("token") or data.get("access_token") or data.get("data", {}).get("token")
    return None


def create_test_user(http_client, api_url, admin_token, username, role="user"):
    """Cria um usuário de teste"""
    headers = {"Authorization": f"Bearer {admin_token}"}
    response = http_client.post(f"{api_url}/users", json={
        "username": username,
        "display_name": f"Test {username}",
        "password": DEFAULT_STRONG_PASSWORD,
        "role": role
    }, headers=headers)

    if response.status_code in [200, 201]:
        data = response.json()
        return {
            "id": data.get("id") or data.get("data", {}).get("id"),
            "username": username,
            "password": DEFAULT_STRONG_PASSWORD,
            "role": role
        }
    return None
