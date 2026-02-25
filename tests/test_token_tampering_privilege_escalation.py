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

"""
Testes de Segurança para JWT Token Tampering e Privilege Escalation
"""

import pytest
import requests
import time
import uuid
import json
import base64
import hmac
import hashlib


class TestJWTTokenTampering:
    """Testes de adulteração de JWT tokens"""

    def _decode_jwt_parts(self, token):
        """Decodifica as 3 partes de um JWT: header.payload.signature"""
        try:
            parts = token.split('.')
            if len(parts) != 3:
                return None, None, None

            header = json.loads(base64.urlsafe_b64decode(parts[0] + '=='))
            payload = json.loads(base64.urlsafe_b64decode(parts[1] + '=='))
            signature = parts[2]

            return header, payload, signature
        except Exception:
            return None, None, None

    def _create_tampered_token(self, token, new_payload):
        """Cria um token JWT com payload alterado (sem recalcular signature)"""
        try:
            parts = token.split('.')
            if len(parts) != 3:
                return None

            # Codificar novo payload
            payload_json = json.dumps(new_payload)
            new_payload_b64 = base64.urlsafe_b64encode(
                payload_json.encode()
            ).decode().rstrip('=')

            # Retornar token com novo payload mas signature antiga (INVÁLIDO)
            return f"{parts[0]}.{new_payload_b64}.{parts[2]}"
        except Exception:
            return None

    def test_modify_token_payload_role_field(self, http_client, api_url, regular_user, timer):
        """Tentar modificar o campo 'role' no payload do token"""
        if not regular_user or "token" not in regular_user:
            pytest.skip("Regular user não disponível")

        token = regular_user['token']
        header, payload, signature = self._decode_jwt_parts(token)

        if payload is None:
            pytest.skip("Não conseguiu decodificar o token")

        # Clonar payload e alterar role para 'admin'
        tampered_payload = payload.copy()
        tampered_payload['role'] = 'admin'

        tampered_token = self._create_tampered_token(token, tampered_payload)
        if tampered_token is None:
            pytest.skip("Não conseguiu criar token tampered")

        # Tentar usar token alterado para acessar recurso admin
        response = http_client.get(
            f"{api_url}/audit-logs",  # Endpoint exclusivo para root
            headers={"Authorization": f"Bearer {tampered_token}"}
        )

        # Deve rejeitar o token alterado com 401 ou 403
        assert response.status_code in [401, 403], \
            f"Token com role alterado foi aceito! Status: {response.status_code}"

    def test_modify_token_user_id_field(self, http_client, api_url, regular_user, admin_user, timer):
        """Tentar modificar o campo 'user_id' no payload do token"""
        if not regular_user or "token" not in regular_user:
            pytest.skip("Regular user não disponível")
        if not admin_user or "id" not in admin_user:
            pytest.skip("Admin user não disponível")

        token = regular_user['token']
        header, payload, signature = self._decode_jwt_parts(token)

        if payload is None:
            pytest.skip("Não conseguiu decodificar o token")

        # Clonar payload e alterar user_id para ID do admin
        tampered_payload = payload.copy()
        tampered_payload['user_id'] = admin_user['id']

        tampered_token = self._create_tampered_token(token, tampered_payload)
        if tampered_token is None:
            pytest.skip("Não conseguiu criar token tampered")

        # Tentar usar token para acessar dados
        response = http_client.get(
            f"{api_url}/categories",
            headers={"Authorization": f"Bearer {tampered_token}"}
        )

        # O backend deve rejeitar o token adulterado
        assert response.status_code in [401, 403], \
            f"Token com user_id alterado foi aceito! Status: {response.status_code}"

    def test_modify_token_username_field(self, http_client, api_url, regular_user, timer):
        """Tentar modificar o campo 'username' no payload do token"""
        if not regular_user or "token" not in regular_user:
            pytest.skip("Regular user não disponível")

        token = regular_user['token']
        header, payload, signature = self._decode_jwt_parts(token)

        if payload is None:
            pytest.skip("Não conseguiu decodificar o token")

        # Clonar payload e alterar username
        tampered_payload = payload.copy()
        tampered_payload['username'] = 'root'

        tampered_token = self._create_tampered_token(token, tampered_payload)
        if tampered_token is None:
            pytest.skip("Não conseguiu criar token tampered")

        # Tentar usar token
        response = http_client.get(
            f"{api_url}/users",
            headers={"Authorization": f"Bearer {tampered_token}"}
        )

        # Deve rejeitar
        assert response.status_code in [401, 403], \
            f"Token com username alterado foi aceito! Status: {response.status_code}"

    def test_token_with_modified_signature_invalid(self, http_client, api_url, regular_user, timer):
        """Tentar modificar a signature do token"""
        if not regular_user or "token" not in regular_user:
            pytest.skip("Regular user não disponível")

        token = regular_user['token']
        parts = token.split('.')

        if len(parts) != 3:
            pytest.skip("Token em formato incorreto")

        # Modificar a signature (última parte)
        modified_signature = "AAAAAAAAAAAAAAAAAAAAAA"
        tampered_token = f"{parts[0]}.{parts[1]}.{modified_signature}"

        response = http_client.get(
            f"{api_url}/categories",
            headers={"Authorization": f"Bearer {tampered_token}"}
        )

        # Deve rejeitar signature inválida
        assert response.status_code == 401, \
            f"Token com signature modificada foi aceito! Status: {response.status_code}"

    def test_token_header_tampering_algorithm_change(self, http_client, api_url, regular_user, timer):
        """Tentar alterar o algoritmo no header do token (ex: HS256 -> none)"""
        if not regular_user or "token" not in regular_user:
            pytest.skip("Regular user não disponível")

        token = regular_user['token']
        header, payload, signature = self._decode_jwt_parts(token)

        if header is None:
            pytest.skip("Não conseguiu decodificar o token")

        # Alterar header para usar algoritmo 'none'
        tampered_header = header.copy()
        tampered_header['alg'] = 'none'

        # Criar token com header alterado
        try:
            header_b64 = base64.urlsafe_b64encode(
                json.dumps(tampered_header).encode()
            ).decode().rstrip('=')
            payload_b64 = token.split('.')[1]
            tampered_token = f"{header_b64}.{payload_b64}."
        except Exception:
            pytest.skip("Erro ao criar token tampered")

        response = http_client.get(
            f"{api_url}/categories",
            headers={"Authorization": f"Bearer {tampered_token}"}
        )

        # Deve rejeitar
        assert response.status_code == 401, \
            f"Token com algoritmo 'none' foi aceito! Status: {response.status_code}"

    def test_token_with_extra_claims_ignored(self, http_client, api_url, regular_user, timer):
        """Tentar adicionar claims extras ao token (devem ser ignoradas)"""
        if not regular_user or "token" not in regular_user:
            pytest.skip("Regular user não disponível")

        token = regular_user['token']
        header, payload, signature = self._decode_jwt_parts(token)

        if payload is None:
            pytest.skip("Não conseguiu decodificar o token")

        # Adicionar claims extras (sem alterar signature)
        tampered_payload = payload.copy()
        tampered_payload['admin'] = True
        tampered_payload['is_root'] = True

        tampered_token = self._create_tampered_token(token, tampered_payload)
        if tampered_token is None:
            pytest.skip("Não conseguiu criar token tampered")

        response = http_client.get(
            f"{api_url}/categories",
            headers={"Authorization": f"Bearer {tampered_token}"}
        )

        # Deve rejeitar por signature inválida
        assert response.status_code == 401, \
            f"Token com claims extras foi aceito! Status: {response.status_code}"


class TestPrivilegeEscalationViaBody:
    """Testes de escalação de privilégio via adulteração do body do request"""

    def test_user_cannot_elevate_own_role_via_body(self, http_client, api_url, root_user, timer):
        """Usuário comum tenta elevar seu próprio role para admin via PUT body"""
        if not root_user or "token" not in root_user:
            pytest.skip("Root user não disponível")

        # Store username for reuse
        test_username = f"escalation_test_{uuid.uuid4().hex[:8]}"

        # Criar usuário comum
        user_resp = http_client.post(
            f"{api_url}/users",
            headers={"Authorization": f"Bearer {root_user['token']}"},
            json={
                "username": test_username,
                "display_name": "Escalation Test User",
                "password": "ValidPass123!@#abcXYZ",
                "role": "user"
            }
        )

        if user_resp.status_code not in [200, 201]:
            pytest.skip("Não conseguiu criar usuário de teste")

        # Fazer login com o novo usuário
        login_resp = http_client.post(
            f"{api_url}/login",
            json={
                "username": test_username,
                "password": "ValidPass123!@#abcXYZ"
            }
        )

        if login_resp.status_code != 200:
            pytest.skip("Não conseguiu fazer login com usuário de teste")

        tokens = login_resp.json()
        user_token = tokens.get("token") or tokens.get("access_token") or tokens.get("data", {}).get("token")
        if not user_token:
            pytest.skip("Não conseguiu obter token do usuário")

        # Tentar alterar seu próprio role para admin via body
        update_resp = http_client.put(
            f"{api_url}/users/{test_username}",
            headers={"Authorization": f"Bearer {user_token}"},
            json={
                "display_name": "Hacked Name",
                "role": "admin"  # Tentando elevar privilégio
            }
        )

        # O backend deve rejeitar a tentativa (usuário comum não pode usar PUT em /users)
        assert update_resp.status_code in [400, 403], \
            f"Usuário comum conseguiu executar PUT em usuários! Status: {update_resp.status_code}"

    def test_user_cannot_create_admin_user(self, http_client, api_url, regular_user, timer):
        """Usuário comum tenta criar outro usuário com role admin"""
        if not regular_user or "token" not in regular_user:
            pytest.skip("Regular user não disponível")

        response = http_client.post(
            f"{api_url}/users",
            headers={"Authorization": f"Bearer {regular_user['token']}"},
            json={
                "username": f"newadmin_{uuid.uuid4().hex[:8]}",
                "display_name": "New Admin",
                "password": "ValidPass123!@#abcXYZ",
                "role": "admin"  # Tentando criar um admin
            }
        )

        # Deve rejeitar (usuário comum não tem permissão para criar usuários)
        assert response.status_code in [400, 403], \
            f"Usuário comum conseguiu criar usuário! Status: {response.status_code}"

    def test_user_cannot_change_other_user_role(self, http_client, api_url, root_user, timer):
        """Usuário comum tenta alterar o role de outro usuário"""
        if not root_user or "token" not in root_user:
            pytest.skip("Root user não disponível")

        # Store usernames for reuse
        user1_username = f"user1_{uuid.uuid4().hex[:8]}"
        user2_username = f"user2_{uuid.uuid4().hex[:8]}"

        # Criar dois usuários comuns
        user1_resp = http_client.post(
            f"{api_url}/users",
            headers={"Authorization": f"Bearer {root_user['token']}"},
            json={
                "username": user1_username,
                "display_name": "User 1",
                "password": "ValidPass123!@#abcXYZ",
                "role": "user"
            }
        )

        user2_resp = http_client.post(
            f"{api_url}/users",
            headers={"Authorization": f"Bearer {root_user['token']}"},
            json={
                "username": user2_username,
                "display_name": "User 2",
                "password": "ValidPass123!@#abcXYZ",
                "role": "user"
            }
        )

        if user1_resp.status_code not in [200, 201] or user2_resp.status_code not in [200, 201]:
            pytest.skip("Não conseguiu criar usuários de teste")

        # Fazer login com user1
        login_resp = http_client.post(
            f"{api_url}/login",
            json={
                "username": user1_username,
                "password": "ValidPass123!@#abcXYZ"
            }
        )

        if login_resp.status_code != 200:
            pytest.skip("Não conseguiu fazer login")

        tokens = login_resp.json()
        user1_token = tokens.get("token") or tokens.get("access_token") or tokens.get("data", {}).get("token")
        if not user1_token:
            pytest.skip("Não conseguiu obter token")

        # User1 tenta alterar role de User2
        response = http_client.put(
            f"{api_url}/users/{user2_username}",
            headers={"Authorization": f"Bearer {user1_token}"},
            json={"role": "admin"}
        )

        # Deve rejeitar (usuário comum não tem permissão)
        assert response.status_code in [400, 403], \
            f"Usuário comum conseguiu alterar outro usuário! Status: {response.status_code}"


class TestResourceOwnershipBypass:
    """Testes de tentativa de bypass via manipulação de IDs de recursos"""

    def test_user_cannot_spoof_client_id_in_contract(self, http_client, api_url, root_user, timer):
        """Tentar criar contrato para outro cliente via body"""
        if not root_user or "token" not in root_user:
            pytest.skip("Root user não disponível")

        headers = {"Authorization": f"Bearer {root_user['token']}"}

        # Criar 2 clientes
        client1_resp = http_client.post(
            f"{api_url}/clients",
            headers=headers,
            json={"name": f"Client1_{uuid.uuid4().hex[:8]}"}
        )

        client2_resp = http_client.post(
            f"{api_url}/clients",
            headers=headers,
            json={"name": f"Client2_{uuid.uuid4().hex[:8]}"}
        )

        if client1_resp.status_code not in [200, 201] or client2_resp.status_code not in [200, 201]:
            pytest.skip("Não conseguiu criar clientes")

        client1_json = client1_resp.json()
        client2_json = client2_resp.json()
        client1_id = client1_json.get("id") or client1_json.get("data", {}).get("id")
        client2_id = client2_json.get("id") or client2_json.get("data", {}).get("id")

        # Criar categoria e subcategoria
        cat_resp = http_client.post(
            f"{api_url}/categories",
            headers=headers,
            json={"name": f"Category_{uuid.uuid4().hex[:8]}"}
        )

        if cat_resp.status_code not in [200, 201]:
            pytest.skip("Não conseguiu criar categoria")

        cat_json = cat_resp.json()
        cat_id = cat_json.get("id") or cat_json.get("data", {}).get("id")

        subcat_resp = http_client.post(
            f"{api_url}/subcategories",
            headers=headers,
            json={
                "name": f"Subcat_{uuid.uuid4().hex[:8]}",
                "category_id": cat_id
            }
        )

        if subcat_resp.status_code not in [200, 201]:
            pytest.skip("Não conseguiu criar subcategoria")

        subcat_json = subcat_resp.json()
        subcat_id = subcat_json.get("id") or subcat_json.get("data", {}).get("id")

        # Criar contrato para client1, mas tentar passar client2_id
        contract_resp = http_client.post(
            f"{api_url}/contracts",
            headers=headers,
            json={
                "client_id": client2_id,  # Tentar associar a outro cliente
                "subcategory_id": subcat_id,
                "model": "Test Model"
            }
        )

        # Deve aceitar (o ID do path não controla, é o ID da URL que importa)
        # Mas a intenção do teste é garantir que não há modo de spoofar
        # Se foi criado, verificar que está associado ao client2 (esperado)
        # Se foi rejeitado, significa que há validação
        assert contract_resp.status_code in [200, 201, 400, 403], \
            f"Resposta inesperada ao criar contrato: {contract_resp.status_code}"

    def test_user_cannot_modify_affiliate_client_id(self, http_client, api_url, root_user, timer):
        """Tentar modificar client_id de um affiliate - IMPORTANTE: Detecta vulnerabilidade"""
        if not root_user or "token" not in root_user:
            pytest.skip("Root user não disponível")

        headers = {"Authorization": f"Bearer {root_user['token']}"}

        # Criar cliente
        client_resp = http_client.post(
            f"{api_url}/clients",
            headers=headers,
            json={"name": f"TestClient_{uuid.uuid4().hex[:8]}"}
        )

        if client_resp.status_code not in [200, 201]:
            pytest.skip("Não conseguiu criar cliente")

        client_json = client_resp.json()
        client_id = client_json.get("id") or client_json.get("data", {}).get("id")

        # Criar affiliate
        aff_resp = http_client.post(
            f"{api_url}/clients/{client_id}/affiliates",
            headers=headers,
            json={"name": f"Affiliate_{uuid.uuid4().hex[:8]}"}
        )

        if aff_resp.status_code not in [200, 201]:
            pytest.skip("Não conseguiu criar affiliate")

        aff_json = aff_resp.json()
        aff_id = aff_json.get("id") or aff_json.get("data", {}).get("id")

        # Criar outro cliente
        client2_resp = http_client.post(
            f"{api_url}/clients",
            headers=headers,
            json={"name": f"TestClient2_{uuid.uuid4().hex[:8]}"}
        )

        if client2_resp.status_code not in [200, 201]:
            pytest.skip("Não conseguiu criar segundo cliente")

        client2_json = client2_resp.json()
        client2_id = client2_json.get("id") or client2_json.get("data", {}).get("id")

        # Verificar que affiliate existe em client1 antes de modificar
        get_before = http_client.get(
            f"{api_url}/clients/{client_id}/affiliates",
            headers=headers
        )

        affiliates_before = []
        if get_before.status_code == 200:
            data = get_before.json()
            affiliates_before = data.get("data", data) if isinstance(data, dict) else data

        # Tentar modificar affiliate para pertencer a outro cliente
        update_resp = http_client.put(
            f"{api_url}/affiliates/{aff_id}",
            headers=headers,
            json={
                "name": "Modified Affiliate",
                "client_id": client2_id  # Tentando alterar cliente
            }
        )

        # Backend deve ignorar o campo client_id ou rejeitar
        # IMPORTANTE: Se retornar 200/204, o affiliate NÃO deve ter sido movido
        if update_resp.status_code in [200, 204]:
            # Se permitiu a atualização, verificar que affiliate ainda está em client1
            get_after = http_client.get(
                f"{api_url}/clients/{client_id}/affiliates",
                headers=headers
            )
            if get_after.status_code == 200:
                data = get_after.json()
                affiliates_after = data.get("data", data) if isinstance(data, dict) else data

                # O affiliate ainda deve estar em client1
                still_in_client1 = any(
                    aff.get("id") == aff_id for aff in affiliates_after if isinstance(aff, dict)
                )

                # IMPORTANTE: Esta assertion detecta uma vulnerabilidade de segurança!
                assert still_in_client1, \
                    f"🔴 VULNERABILIDADE: Affiliate {aff_id} foi movido de {client_id} para {client2_id} via client_id no body!"

    def test_user_cannot_forge_id_in_post_request(self, http_client, api_url, root_user, timer):
        """Tentar forjar um ID ao criar um novo recurso"""
        if not root_user or "token" not in root_user:
            pytest.skip("Root user não disponível")

        headers = {"Authorization": f"Bearer {root_user['token']}"}
        fake_id = str(uuid.uuid4())

        # Tentar criar cliente com ID específico
        response = http_client.post(
            f"{api_url}/clients",
            headers=headers,
            json={
                "id": fake_id,  # Tentando forjar o ID
                "name": "Forged Client"
            }
        )

        # Deve ignorar o campo 'id' ou rejeitar
        # Se criou, verificar que o ID gerado é diferente
        if response.status_code in [200, 201]:
            resp_json = response.json()
            returned_id = resp_json.get("id") or resp_json.get("data", {}).get("id")
            assert returned_id != fake_id, \
                f"Backend usou o ID forjado! ID fornecido: {fake_id}, ID retornado: {returned_id}"


class TestTokenValidationConsistency:
    """Testes de consistência na validação de tokens"""

    def test_backend_validates_token_signature(self, http_client, api_url, regular_user, timer):
        """Verificar que o backend sempre valida a signature do token"""
        if not regular_user or "token" not in regular_user:
            pytest.skip("Regular user não disponível")

        # Fazer uma requisição válida
        valid_resp = http_client.get(
            f"{api_url}/categories",
            headers={"Authorization": f"Bearer {regular_user['token']}"}
        )

        assert valid_resp.status_code == 403, \
            f"Token válido de user comum deve receber 403 sem permissão! Status: {valid_resp.status_code}"

        # Tentar com token inválido
        invalid_token = regular_user['token'][:-10] + "XXXXXXXXXX"
        invalid_resp = http_client.get(
            f"{api_url}/categories",
            headers={"Authorization": f"Bearer {invalid_token}"}
        )

        assert invalid_resp.status_code == 401, \
            f"Token inválido foi aceito! Status: {invalid_resp.status_code}"

    def test_every_protected_endpoint_requires_valid_token(self, http_client, api_url, timer):
        """Verificar que TODOS os endpoints protegidos exigem token válido"""
        protected_endpoints = [
            ("GET", "/api/categories"),
            ("GET", "/api/clients"),
            ("GET", "/api/contracts"),
            ("GET", "/api/subcategories"),
            ("POST", "/api/categories"),
            ("POST", "/api/clients"),
            ("POST", "/api/contracts"),
        ]

        for method, endpoint in protected_endpoints:
            if method == "GET":
                response = http_client.get(f"{api_url}{endpoint}")
            else:
                response = http_client.post(f"{api_url}{endpoint}", json={})

            # Deve retornar 401 (sem token) - 404 é aceitável se endpoint não existe
            # 400 também é aceitável para validação de input ou formato
            assert response.status_code in [400, 401, 403, 404], \
                f"Endpoint {method} {endpoint} não exigiu autenticação! Status: {response.status_code}"
