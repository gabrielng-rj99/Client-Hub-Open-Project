/*
 * This file is part of Client Hub Open Project.
 * Copyright (C) 2025 Client Hub Contributors
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

import { handleResponseErrors } from "./apiHelpers";

export const contractsApi = {
    // Direct return method for getting contracts (used by AuditLogs)
    getContracts: async (apiUrl, token, params = {}, onTokenExpired) => {
        const queryParams = new URLSearchParams();
        if (params.limit) queryParams.append("limit", params.limit);

        const url = `${apiUrl}/contracts${queryParams.toString() ? "?" + queryParams.toString() : ""}`;

        const response = await fetch(url, {
            headers: {
                Authorization: `Bearer ${token}`,
                "Content-Type": "application/json",
            },
        });

        handleResponseErrors(response, onTokenExpired);

        if (!response.ok) {
            throw new Error("Erro ao carregar contratos");
        }

        return await response.json();
    },

    loadContracts: async (apiUrl, token, onTokenExpired) => {
        const response = await fetch(`${apiUrl}/contracts`, {
            headers: {
                Authorization: `Bearer ${token}`,
                "Content-Type": "application/json",
            },
        });

        handleResponseErrors(response, onTokenExpired);

        if (!response.ok) {
            throw new Error("Erro ao carregar contratos");
        }

        const data = await response.json();
        return data.data || [];
    },

    loadClients: async (apiUrl, token, onTokenExpired) => {
        const response = await fetch(`${apiUrl}/clients`, {
            headers: {
                Authorization: `Bearer ${token}`,
                "Content-Type": "application/json",
            },
        });

        handleResponseErrors(response, onTokenExpired);

        if (!response.ok) {
            throw new Error("Erro ao carregar clientes");
        }

        const data = await response.json();
        return data.data || [];
    },

    loadCategories: async (apiUrl, token, onTokenExpired) => {
        const response = await fetch(`${apiUrl}/categories`, {
            headers: {
                Authorization: `Bearer ${token}`,
                "Content-Type": "application/json",
            },
        });

        handleResponseErrors(response, onTokenExpired);

        if (!response.ok) {
            throw new Error("Erro ao carregar categorias");
        }

        const data = await response.json();
        return data.data || [];
    },

    loadSubcategories: async (apiUrl, token, categoryId, onTokenExpired) => {
        const response = await fetch(
            `${apiUrl}/categories/${categoryId}/subcategories`,
            {
                headers: {
                    Authorization: `Bearer ${token}`,
                    "Content-Type": "application/json",
                },
            },
        );

        handleResponseErrors(response, onTokenExpired);

        if (!response.ok) {
            throw new Error("Erro ao carregar linhas");
        }

        const data = await response.json();
        return data.data || [];
    },

    loadAffiliates: async (apiUrl, token, clientId, onTokenExpired) => {
        const response = await fetch(
            `${apiUrl}/clients/${clientId}/affiliates`,
            {
                headers: {
                    Authorization: `Bearer ${token}`,
                    "Content-Type": "application/json",
                },
            },
        );

        handleResponseErrors(response, onTokenExpired);

        if (!response.ok) {
            throw new Error("Erro ao carregar afiliados");
        }

        const data = await response.json();
        return data.data || [];
    },

    createContract: async (apiUrl, token, formData, onTokenExpired) => {
        const payload = {
            model: formData.model || "",
            item_key: formData.item_key || "",
            start_date: formData.start_date,
            end_date: formData.end_date,
            client_id: formData.client_id,
            affiliate_id: formData.affiliate_id || null,
            subcategory_id: formData.subcategory_id,
        };

        const response = await fetch(`${apiUrl}/contracts`, {
            method: "POST",
            headers: {
                Authorization: `Bearer ${token}`,
                "Content-Type": "application/json",
            },
            body: JSON.stringify(payload),
        });

        handleResponseErrors(response, onTokenExpired);

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || "Erro ao criar contrato");
        }

        return response.json();
    },

    updateContract: async (
        apiUrl,
        token,
        contractId,
        formData,
        onTokenExpired,
    ) => {
        const payload = {
            model: formData.model || "",
            item_key: formData.item_key || "",
            start_date: formData.start_date,
            end_date: formData.end_date,
            client_id: formData.client_id,
            affiliate_id: formData.affiliate_id || null,
            subcategory_id: formData.subcategory_id,
        };

        const response = await fetch(`${apiUrl}/contracts/${contractId}`, {
            method: "PUT",
            headers: {
                Authorization: `Bearer ${token}`,
                "Content-Type": "application/json",
            },
            body: JSON.stringify(payload),
        });

        handleResponseErrors(response, onTokenExpired);

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || "Erro ao atualizar contrato");
        }

        return response.json();
    },

    archiveContract: async (apiUrl, token, contractId, onTokenExpired) => {
        const response = await fetch(
            `${apiUrl}/contracts/${contractId}/archive`,
            {
                method: "PUT",
                headers: {
                    Authorization: `Bearer ${token}`,
                    "Content-Type": "application/json",
                },
            },
        );

        handleResponseErrors(response, onTokenExpired);

        if (!response.ok) {
            throw new Error("Erro ao arquivar contrato");
        }

        return response.json();
    },

    unarchiveContract: async (apiUrl, token, contractId, onTokenExpired) => {
        const response = await fetch(
            `${apiUrl}/contracts/${contractId}/unarchive`,
            {
                method: "PUT",
                headers: {
                    Authorization: `Bearer ${token}`,
                    "Content-Type": "application/json",
                },
            },
        );

        handleResponseErrors(response, onTokenExpired);

        if (!response.ok) {
            throw new Error("Erro ao desarquivar contrato");
        }

        return response.json();
    },



    getContractByID: async (apiUrl, token, contractId, onTokenExpired) => {
        const response = await fetch(`${apiUrl}/contracts/${contractId}`, {
            headers: {
                Authorization: `Bearer ${token}`,
                "Content-Type": "application/json",
            },
        });

        handleResponseErrors(response, onTokenExpired);

        if (!response.ok) {
            throw new Error("Erro ao carregar contrato");
        }

        const data = await response.json();
        return data.data || data;
    },

    // Lazy search for clients in modals — uses existing ?search= param
    searchClients: async (apiUrl, token, query, onTokenExpired, offset = 0) => {
        const params = new URLSearchParams({
            search: query,
            limit: "100",
            offset: offset.toString(),
            include_archived: "false",
        });
        const response = await fetch(`${apiUrl}/clients?${params}`, {
            headers: {
                Authorization: `Bearer ${token}`,
                "Content-Type": "application/json",
            },
        });

        handleResponseErrors(response, onTokenExpired);

        if (!response.ok) {
            throw new Error("Erro ao buscar clientes");
        }

        const data = await response.json();
        return data.data || [];
    },

    searchCategories: async (apiUrl, token, query, onTokenExpired, offset = 0) => {
        const params = new URLSearchParams({
            search: query || "",
            limit: "100",
            offset: offset.toString(),
            include_archived: "false",
        });
        const response = await fetch(`${apiUrl}/categories?${params}`, {
            method: "GET",
            headers: {
                Authorization: `Bearer ${token}`,
                "Content-Type": "application/json",
            },
        });

        handleResponseErrors(response, onTokenExpired);
        if (!response.ok) return [];

        const data = await response.json();
        return data.data || [];
    },
};
