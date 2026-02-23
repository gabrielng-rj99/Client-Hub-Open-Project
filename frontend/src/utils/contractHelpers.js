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

// Helper para converter data do formato yyyy-mm-dd para dd/mm/yyyy
const convertDateToDisplay = (dateString) => {
    if (!dateString) return "";
    const [year, month, day] = dateString.split("T")[0].split("-");
    return `${day}/${month}/${year}`;
};

// Helper para converter data do formato dd/mm/yyyy para yyyy-mm-ddT00:00:00Z
const convertDateToAPI = (dateString) => {
    if (!dateString) return "";
    const parts = dateString.split("/");
    if (parts.length === 3) {
        const [day, month, year] = parts;
        if (year && year.length === 4) {
            return `${year}-${month.padStart(2, "0")}-${day.padStart(2, "0")}T00:00:00Z`;
        }
    }
    // If already in YYYY-MM-DD format, add time
    if (dateString.match(/^\d{4}-\d{2}-\d{2}$/)) {
        return `${dateString}T00:00:00Z`;
    }
    return dateString;
};

export const getInitialFormData = () => ({
    model: "",
    item_key: "",
    start_date: "",
    end_date: "",
    client_id: "",
    affiliate_id: "",
    category_id: "",
    subcategory_id: "",
});

export const formatContractForEdit = (contract) => ({
    model: contract.model || "",
    item_key: contract.item_key || "",
    // Store as yyyy-mm-dd for native <input type="date"> compatibility
    start_date: contract.start_date
        ? contract.start_date.split("T")[0]   // strip time zone portion
        : "",
    end_date: contract.end_date
        ? contract.end_date.split("T")[0]
        : "",
    client_id: contract.client_id || "",
    affiliate_id: contract.affiliate_id || "",
    // category_id: prefer the embedded field, but ContractsModal will
    // also derive it from categories × subcategory_id as a fallback.
    category_id: contract.line?.category_id || contract.category_id || "",
    subcategory_id: contract.subcategory_id || "",
});

export const formatDate = (dateString) => {
    if (!dateString) return "-";
    // Parse date correctly to avoid timezone issues
    // If format is yyyy-mm-dd, parse manually
    if (dateString.match(/^\d{4}-\d{2}-\d{2}/)) {
        const [year, month, day] = dateString.split("T")[0].split("-");
        return `${day}/${month}/${year}`;
    }
    try {
        const date = new Date(dateString);
        return date.toLocaleDateString("pt-BR", {
            day: "2-digit",
            month: "2-digit",
            year: "numeric",
        });
    } catch {
        return "-";
    }
};

export const getContractStatus = (contract) => {
    // Se não tem data de término, é considerado ativo (acordo permanente)
    if (!contract.end_date || contract.end_date === "") {
        return { status: "Ativo", color: "#27ae60" };
    }

    const endDate = new Date(contract.end_date);
    const startDate = contract.start_date
        ? new Date(contract.start_date)
        : null;
    const now = new Date();

    // Se tem data de início no futuro, é "Não Iniciado"
    if (startDate && startDate > now) {
        return { status: "Não Iniciado", color: "#3498db" };
    }

    const daysUntilExpiration = Math.ceil(
        (endDate - now) / (1000 * 60 * 60 * 24),
    );

    if (daysUntilExpiration < 0) {
        return { status: "Expirado", color: "#e74c3c" };
    } else if (daysUntilExpiration <= 30) {
        return { status: "Próximo ao vencimento", color: "#f39c12" };
    } else {
        return { status: "Ativo", color: "#27ae60" };
    }
};

export const filterContracts = (
    contracts,
    filter,
    searchFilters,
    clients,
    categories,
    lookupMaps = null,
) => {
    // searchFilters can be an object with: categoryId, subcategoryId, clientName, contractName
    // or a string for backward compatibility
    const filters =
        typeof searchFilters === "object"
            ? searchFilters
            : { searchTerm: searchFilters };

    const clientById =
        lookupMaps?.clientById ||
        new Map(clients.map((client) => [client.id, client]));

    const categoryBySubcategoryId =
        lookupMaps?.categoryBySubcategoryId ||
        (() => {
            const map = new Map();
            for (const category of categories) {
                if (!Array.isArray(category.lines)) continue;
                for (const line of category.lines) {
                    map.set(line.id, category);
                }
            }
            return map;
        })();

    return contracts.filter((contract) => {
        const isArchived = !!contract.archived_at;
        const statusObj = getContractStatus(contract);
        const status = statusObj.status;

        const matchesFilter =
            (filter === "active" && !isArchived && status === "Ativo") ||
            (filter === "expiring" &&
                !isArchived &&
                status === "Próximo ao vencimento") ||
            (filter === "expired" && !isArchived && status === "Expirado") ||
            (filter === "not-started" &&
                !isArchived &&
                status === "Não Iniciado") ||
            (filter === "archived" && isArchived) ||
            (filter === "all" && !isArchived);

        if (!matchesFilter) return false;

        // Use enriched fields when available, fallback to map lookup
        const client = contract.client_name
            ? { name: contract.client_name }
            : clientById.get(contract.client_id);
        const category = contract.category_name
            ? { id: null, name: contract.category_name }
            : categoryBySubcategoryId.get(contract.subcategory_id);

        // Filter by category ID
        if (filters.categoryId && filters.categoryId !== "") {
            // When using enriched data, we need map lookup for ID matching
            const catFromMap = categoryBySubcategoryId.get(
                contract.subcategory_id,
            );
            if (!catFromMap || catFromMap.id !== filters.categoryId) {
                return false;
            }
        }

        // Filter by subcategory ID
        if (filters.subcategoryId && filters.subcategoryId !== "") {
            if (contract.subcategory_id !== filters.subcategoryId) {
                return false;
            }
        }

        // Filter by client name (partial match) — use enriched field
        if (filters.clientName && filters.clientName !== "") {
            const clientNameLower = filters.clientName.toLowerCase();
            const resolvedName = contract.client_name || client?.name || "";
            if (!resolvedName.toLowerCase().includes(clientNameLower)) {
                return false;
            }
        }

        // Filter by contract name/model (partial match)
        if (filters.contractName && filters.contractName !== "") {
            const contractNameLower = filters.contractName.toLowerCase();
            const matchesModel = contract.model
                ?.toLowerCase()
                .includes(contractNameLower);
            const matchesItemKey = contract.item_key
                ?.toLowerCase()
                .includes(contractNameLower);
            if (!matchesModel && !matchesItemKey) {
                return false;
            }
        }

        // Legacy search term support (searches all fields)
        if (filters.searchTerm && filters.searchTerm !== "") {
            const searchLower = filters.searchTerm.toLowerCase();
            const resolvedClientName =
                contract.client_name || client?.name || "";
            const resolvedCategoryName =
                contract.category_name || category?.name || "";
            const matchesSearch =
                contract.model?.toLowerCase().includes(searchLower) ||
                contract.item_key?.toLowerCase().includes(searchLower) ||
                resolvedClientName.toLowerCase().includes(searchLower) ||
                resolvedCategoryName.toLowerCase().includes(searchLower);
            if (!matchesSearch) {
                return false;
            }
        }

        return true;
    });
};

export const getClientName = (clientIdOrContract, clients) => {
    // Support enriched contract object directly
    if (
        typeof clientIdOrContract === "object" &&
        clientIdOrContract?.client_name
    ) {
        return clientIdOrContract.client_name;
    }
    const clientId =
        typeof clientIdOrContract === "string"
            ? clientIdOrContract
            : clientIdOrContract?.client_id;
    const client = clients.find((c) => c.id === clientId);
    return client?.name || "-";
};

export const getCategoryName = (lineId, categories) => {
    const category = categories.find((cat) =>
        cat.lines?.some((line) => line.id === lineId),
    );
    return category?.name || "-";
};

// Build the API payload from formData.
// Handles both yyyy-mm-dd (native date input) and dd/mm/yyyy (legacy).
export const prepareContractDataForAPI = (formData) => {
    const toAPIDate = (val) => {
        if (!val) return null;
        // dd/mm/yyyy → yyyy-mm-ddT00:00:00Z
        if (/^\d{2}\/\d{2}\/\d{4}$/.test(val)) {
            return convertDateToAPI(val);
        }
        // yyyy-mm-dd → yyyy-mm-ddT00:00:00Z
        if (/^\d{4}-\d{2}-\d{2}$/.test(val)) {
            return `${val}T00:00:00Z`;
        }
        return null;
    };
    return {
        ...formData,
        start_date: toAPIDate(formData.start_date),
        end_date: toAPIDate(formData.end_date),
    };
};
