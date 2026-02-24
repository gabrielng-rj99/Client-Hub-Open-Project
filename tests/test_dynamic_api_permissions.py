import pytest
import uuid

# Define the exhaustive list of endpoints and their required permissions
# Format: (resource, action, method, path, dummy_payload)
API_ENDPOINTS = [
    # Settings
    ("settings", "read", "GET", "/api/settings", None),
    ("settings", "update", "PUT", "/api/settings", {"company_name": "Test"}),
    ("settings", "manage_security", "PUT", "/api/settings/security", {"session_timeout": 60}),
    ("settings", "manage_branding", "PUT", "/api/settings/global-theme", {"preset": "dark"}),
    ("settings", "view_system_info", "GET", "/api/settings/system-config", None),
    
    # Roles
    ("roles", "create", "POST", "/api/roles", {"name": "test_role", "display_name": "Test", "priority": 10}),
    ("roles", "read", "GET", "/api/roles", None),
    ("roles", "update", "PUT", "/api/roles/{id}", {"display_name": "Updated"}),
    ("roles", "delete", "DELETE", "/api/roles/{id}", None),
    ("roles", "manage_permissions", "PUT", "/api/roles/{id}/permissions", {"permission_ids": []}),
    
    # Theme API endpoints omitted as they use hardcoded boolean system settings
    

    # Dashboard
    ("dashboard", "read", "GET", "/api/dashboard/counts", None),
    ("dashboard", "configure", "PUT", "/api/system-config/dashboard", {"show_birthdays": True}),
    
    # Users
    ("users", "create", "POST", "/api/users", {"username": "testpost", "password": "P@ssword123", "role": "user"}),
    ("users", "read", "GET", "/api/users", None),
    ("users", "update", "PUT", "/api/users/{id}", {"display_name": "New Name"}),
    ("users", "delete", "DELETE", "/api/users/{id}", None),
    ("users", "block", "POST", "/api/users/{id}/block", None),
    
    # Clients
    ("clients", "create", "POST", "/api/clients", {"name": "Test Client"}),
    ("clients", "read", "GET", "/api/clients", None),
    ("clients", "update", "PUT", "/api/clients/{id}", {"name": "Update"}),
    ("clients", "delete", "DELETE", "/api/clients/{id}", None),
    ("clients", "archive", "POST", "/api/clients/{id}/archive", None),
    
    # Affiliates
    ("affiliates", "create", "POST", "/api/clients/{id}/affiliates", {"name": "Test Affiliate", "client_id": "{id}"}),
    ("affiliates", "read", "GET", "/api/affiliates/{id}", None),
    ("affiliates", "update", "PUT", "/api/affiliates/{id}", {"name": "Updated"}),
    ("affiliates", "delete", "DELETE", "/api/affiliates/{id}", None),
    
    # Categories
    ("categories", "create", "POST", "/api/categories", {"name": "Test Category"}),
    ("categories", "read", "GET", "/api/categories", None),
    ("categories", "update", "PUT", "/api/categories/{id}", {"name": "Updated"}),
    ("categories", "delete", "DELETE", "/api/categories/{id}", None),
    
    # Subcategories
    ("subcategories", "create", "POST", "/api/subcategories", {"name": "Sub", "category_id": "{id}"}),
    ("subcategories", "read", "GET", "/api/subcategories", None),
    ("subcategories", "update", "PUT", "/api/subcategories/{id}", {"name": "Updated"}),
    ("subcategories", "delete", "DELETE", "/api/subcategories/{id}", None),
    
    # Contracts
    ("contracts", "create", "POST", "/api/contracts", {"contract_model": "test", "client_id": "{id}", "subcategory_id": "{id}"}),
    ("contracts", "read", "GET", "/api/contracts", None),
    ("contracts", "update", "PUT", "/api/contracts/{id}", {"contract_model": "test2"}),
    ("contracts", "delete", "DELETE", "/api/contracts/{id}", None),
    ("contracts", "archive", "POST", "/api/contracts/{id}/archive", None),
    
    # Audit Logs
    ("audit_logs", "read", "GET", "/api/audit-logs", None),
    ("audit_logs", "export", "GET", "/api/audit-logs/export", None),
    ("audit_logs", "delete", "DELETE", "/api/audit-logs/{id}", None),
    
    # Financial
    ("financial", "create", "POST", "/api/financial", {"contract_id": "{id}", "financial_type": "unico"}),
    ("financial", "read", "GET", "/api/financial", None),
    ("financial", "update", "PUT", "/api/financial/{id}", {"financial_type": "mensal"}),
    ("financial", "delete", "DELETE", "/api/financial/{id}", None),
    ("financial", "read_values", "GET", "/api/financial/summary", None),
]

@pytest.fixture(scope="module")
def dynamic_test_role(http_client, api_url, root_user):
    """Cria uma role customizada para os testes."""
    if not root_user or "token" not in root_user:
        pytest.skip("Root token required")
    
    headers = {"Authorization": f"Bearer {root_user['token']}"}
    role_payload = {
        "name": "dynamic_test_role",
        "display_name": "Dynamic Test Role",
        "priority": 5,
        "description": "Role for dynamic API testing"
    }
    resp = http_client.post(f"{api_url}/roles", json=role_payload, headers=headers)
    
    data = resp.json()
    role_id = data.get("id") or data.get("data", {}).get("id")
    if not role_id:
        # Tenta buscar a role se ela já existir
        roles_resp = http_client.get(f"{api_url}/roles", headers=headers)
        for r in roles_resp.json():
            if r.get("role", {}).get("name") == "dynamic_test_role":
                role_id = r["role"]["id"]
                break
            elif r.get("name") == "dynamic_test_role":
                role_id = r["id"]
                break

    assert role_id is not None, "Falha ao criar/encontrar role de teste"

    yield role_id

    # Cleanup
    http_client.delete(f"{api_url}/roles/{role_id}", headers=headers)

@pytest.fixture(scope="module")
def all_permissions_map(http_client, api_url, root_user):
    """Busca os IDs reais de todas as permissões no banco via API."""
    headers = {"Authorization": f"Bearer {root_user['token']}"}
    resp = http_client.get(f"{api_url}/permissions", headers=headers)
    assert resp.status_code == 200, "Falha ao carregar permissoes"
    
    perm_map = {}
    data = resp.json()
    # /permissions retorna um array agrupado ou unflatten
    if isinstance(data, dict):
        for category, perms in data.items():
            for p in perms:
                perm_map[f"{p['resource']}:{p['action']}"] = p['id']
    else:
        for p in data:
            perm_map[f"{p['resource']}:{p['action']}"] = p['id']
            
    return perm_map

@pytest.fixture(scope="module")
def custom_test_user(http_client, api_url, root_user, dynamic_test_role):
    """Cria um usuário com a role customizada."""
    headers = {"Authorization": f"Bearer {root_user['token']}"}
    username = f"dyntest_user_{uuid.uuid4().hex[:8]}"
    pwd = "ValidPass123!@##"
    
    user_payload = {
        "username": username,
        "password": pwd,
        "display_name": "Dynamic Tester",
        "role": "dynamic_test_role"  # API expects role name here
    }
    
    resp = http_client.post(f"{api_url}/users", json=user_payload, headers=headers)
    assert resp.status_code in [200, 201], f"Failed to create user: {resp.text}"
    
    # Login to get token
    login_resp = http_client.post(f"{api_url}/login", json={"username": username, "password": pwd})
    assert login_resp.status_code == 200
    
    token = login_resp.json().get("token") or login_resp.json().get("access_token") or login_resp.json().get("data", {}).get("token")
    user_id = login_resp.json().get("user_id") or login_resp.json().get("data", {}).get("user_id")
    assert token is not None, "Failed to get token for custom tester"
    
    yield {"user_id": user_id, "token": token, "username": username}

@pytest.mark.security
@pytest.mark.parametrize("resource, action, method, path, payload", API_ENDPOINTS)
def test_dynamic_permission_flow(
    http_client, api_url, root_user, dynamic_test_role,
    custom_test_user, all_permissions_map,
    resource, action, method, path, payload
):
    """
    Roda os 4 cenários para um endpoint/ação específica.
    """
    if f"{resource}:{action}" not in all_permissions_map:
        pytest.skip(f"Permissão {resource}:{action} não encontrada no banco (skip).")
        
    perm_id = all_permissions_map[f"{resource}:{action}"]
    root_auth = {"Authorization": f"Bearer {root_user['token']}"}
    user_auth = {"Authorization": f"Bearer {custom_test_user['token']}"}
    
    # Formata URL mockada
    mock_id = str(uuid.uuid4())
    clean_path = path[4:] if path.startswith('/api') else path
    url = f"{api_url}{clean_path.replace('{id}', mock_id)}"
    
    # Função auxiliar para fazer o request com o custom user
    def make_user_request():
        kwargs = {"headers": user_auth}
        if payload is not None:
            # Deep copy or format payload
            p = dict(payload)
            for k, v in p.items():
                if isinstance(v, str) and "{id}" in v:
                    p[k] = v.replace("{id}", mock_id)
            kwargs["json"] = p
            
        return http_client.request(method, url, **kwargs)

    # Função auxiliar para setar permissão
    def set_permissions(perm_list):
        return http_client.put(
            f"{api_url}/roles/{dynamic_test_role}/permissions",
            json={"permission_ids": perm_list},
            headers=root_auth
        )

    # 1. Sem Config (Permissão removida)
    resp_clear = set_permissions([])
    assert resp_clear.status_code in [200, 204], "Falha ao limpar permissoes"
    
    res1 = make_user_request()
    # Se bater num endpoint que n existe, vai dar 404 primeiro no roteador, mas endpoints root param na middleware.
    assert res1.status_code in [401, 403], f"Cenário 1 (Sem config) falhou. Esperado 403, obteve {res1.status_code} ({res1.text}) em {method} {url}"

    # 2. Config Errada (Root tenta injetar erro no PUT de permissões)
    res2 = http_client.put(
        f"{api_url}/roles/{dynamic_test_role}/permissions",
        json={"permission_ids": ["invalid-uuid-string", 1234, {"inject": "true"}]},
        headers=root_auth
    )
    # Exige resposta de Bad Request ou Unprocessable Entity (400 ou 422)
    assert res2.status_code in [400, 422, 500], f"Cenário 2 (Config errada) falhou. Esperado erro 4xx, obteve {res2.status_code}"
    # Wait, se for 500 o backend está vulnerável a crash, mas para o script de segurança exigimos 400!
    assert res2.status_code != 500, "Cenário 2 detectou ERRO 500 Interno (Vulnerabilidade de Validação). Esperado 400."

    # 3. Permitido (Atribui apenas a permissão requerida)
    res_set = set_permissions([perm_id])
    assert res_set.status_code in [200, 204], "Falha ao atribuir permissão"
    
    res3 = make_user_request()
    if res3.status_code == 403:
        txt = res3.text.lower()
        err_msg = f"Cenário 3 falhou. Permissão ({perm_id}) obteve 403 na url {url}: {res3.text}"
        assert "root" in txt or "admin" in txt, err_msg
    else:
        assert res3.status_code not in [401, 403], f"Cenário 3 (Permitido) falhou. Permissão atribuída ({perm_id}) mas obteve {res3.status_code} em {method} {url}"

    # 4. Não Permitido (Remove novamente)
    res_remove = set_permissions([])
    assert res_remove.status_code in [200, 204], "Falha ao remover permissao"
    
    res4 = make_user_request()
    assert res4.status_code in [401, 403], f"Cenário 4 (Não Permitido) falhou. Permissão limpa mas obteve {res4.status_code} em {method} {url}"
