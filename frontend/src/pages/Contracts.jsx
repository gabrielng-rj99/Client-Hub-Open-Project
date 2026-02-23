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

import React, { useState, useEffect, useRef, useMemo } from "react";
import Select from "react-select";
import AsyncSelect from "react-select/async";
import { useConfig } from "../contexts/ConfigContext";
import { useData } from "../contexts/DataContext";
import { contractsApi } from "../api/contractsApi";
import { financialApi } from "../api/financialApi";
import { useUrlState } from "../hooks/useUrlState";
import {
    getInitialFormData,
    formatContractForEdit,
    filterContracts,
    formatDate,
    getContractStatus,
    getClientName,
    prepareContractDataForAPI,
} from "../utils/contractHelpers";
import ContractsTable from "../components/contracts/ContractsTable";
import ContractModal from "../components/contracts/ContractsModal";
import PrimaryButton from "../components/common/PrimaryButton";
import Pagination from "../components/common/Pagination";
import "./styles/Contracts.css";

export default function Contracts({ token, apiUrl, onTokenExpired }) {
    const {
        fetchContracts,
        fetchClients,
        fetchCategories,
        fetchSubcategories,
        createContract,
        updateContract,
        deleteContract,
        invalidateCache,
    } = useData();
    const [contracts, setContracts] = useState([]);
    const [clients, setClients] = useState([]);
    const [categories, setCategories] = useState([]);
    const [lines, setLines] = useState([]);
    const [affiliates, setAffiliates] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");
    const [financialData, setFinancialData] = useState(null);
    const [existingFinancial, setExistingFinancial] = useState(null);

    // Custom styles for react-select to match existing CSS
    const customSelectStyles = {
        control: (provided, state) => ({
            ...provided,
            minWidth: "220px",
            width: "220px",
            fontSize: "14px",
            border: "1px solid var(--border-color, #ddd)",
            borderRadius: "4px",
            background: "var(--content-bg, white)",
            color: "var(--primary-text-color, #333)",
            cursor: "pointer",
            boxShadow: state.isFocused
                ? "0 0 0 1px var(--primary-color, #3498db)"
                : provided.boxShadow,
            "&:hover": {
                borderColor: "var(--primary-color, #3498db)",
            },
        }),
        option: (provided, state) => ({
            ...provided,
            backgroundColor: state.isSelected
                ? "var(--primary-color, #3498db)"
                : state.isFocused
                    ? "var(--hover-bg, #f8f9fa)"
                    : "var(--content-bg, white)",
            color: state.isSelected
                ? "white"
                : "var(--primary-text-color, #333)",
            cursor: "pointer",
        }),
        menu: (provided) => ({
            ...provided,
            background: "var(--content-bg, white)",
            border: "1px solid var(--border-color, #ddd)",
            borderRadius: "4px",
        }),
        singleValue: (provided) => ({
            ...provided,
            color: "var(--primary-text-color, #333)",
        }),
        placeholder: (provided) => ({
            ...provided,
            color: "var(--secondary-text-color, #999)",
        }),
        input: (provided) => ({
            ...provided,
            color: "var(--primary-text-color, #333)",
        }),
    };

    // State persistence
    const { values, updateValue, updateValues, updateValuesImmediate } =
        useUrlState(
            {
                filter: "active",
                page: "1",
                limit: "20",
                categoryId: "",
                subcategoryId: "",
                clientName: "",
                contractName: "",
                sortBy: "end_date",
                sortOrder: "desc",
            },
            { debounce: true, debounceTime: 300, syncWithUrl: false },
        );
    const filter = values.filter;
    const currentPage = parseInt(values.page || "1", 10);
    const itemsPerPage = parseInt(values.limit || "20", 10);
    const categoryIdFilter = values.categoryId || "";
    const subcategoryIdFilter = values.subcategoryId || "";
    const clientNameFilter = values.clientName || "";
    const contractNameFilter = values.contractName || "";
    const sortBy = values.sortBy || "end_date";
    const sortOrder = values.sortOrder || "desc";

    const setFilter = (val) => {
        updateValuesImmediate({ filter: val, page: "1" });
    };
    const setCategoryIdFilter = (val) => {
        updateValuesImmediate({
            categoryId: val,
            subcategoryId: "",
            page: "1",
        });
    };
    const setSubcategoryIdFilter = (val) => {
        updateValuesImmediate({ subcategoryId: val, page: "1" });
    };
    const setClientNameFilter = (val) => {
        updateValues({ clientName: val, page: "1" });
    };
    const setContractNameFilter = (val) => {
        updateValues({ contractName: val, page: "1" });
    };
    const setSortBy = (val) => {
        updateValuesImmediate({ sortBy: val, page: "1" });
    };
    const setSortOrder = (val) => {
        updateValuesImmediate({ sortOrder: val, page: "1" });
    };
    const setCurrentPage = (page) =>
        updateValuesImmediate({ page: page.toString() });
    const setItemsPerPage = (limit) => {
        updateValuesImmediate({ limit: limit.toString(), page: "1" });
    };

    // Get subcategories/lines for selected category
    const availableSubcategories = categoryIdFilter
        ? categories.find((c) => c.id === categoryIdFilter)?.lines || []
        : [];

    const [showModal, setShowModal] = useState(false);
    const [showDetailsModal, setShowDetailsModal] = useState(false);
    const [modalMode, setModalMode] = useState("create");
    const [selectedContract, setSelectedContract] = useState(null);
    const [formData, setFormData] = useState(getInitialFormData());

    // ── Pagination State for Clients Filter ──────────────────────────────────
    const [filterClientOptions, setFilterClientOptions] = useState([]);
    const [filterClientLoading, setFilterClientLoading] = useState(false);
    const [filterClientSearch, setFilterClientSearch] = useState("");
    const [filterClientOffset, setFilterClientOffset] = useState(0);
    const [filterClientHasMore, setFilterClientHasMore] = useState(true);
    const filterClientSearchTimeout = useRef(null);

    const loadFilterClientPage = async (query = "", offset = 0, isNewSearch = false) => {
        setFilterClientLoading(true);
        try {
            const results = await handleSearchClients(query, offset);
            const formatted = results.map(c => ({ value: c.name, label: c.name }));
            const clearOpt = { value: "", label: `Todos os ${config.labels.clients?.toLowerCase() || "clientes"}` };

            if (isNewSearch) {
                setFilterClientOptions([clearOpt, ...formatted]);
            } else {
                setFilterClientOptions(prev => {
                    const existingIds = new Set(prev.map(p => p.value));
                    const newItems = formatted.filter(f => !existingIds.has(f.value));
                    return [...prev, ...newItems];
                });
            }
            setFilterClientHasMore(results.length === 100);
        } catch (e) {
            console.error(e);
        } finally {
            setFilterClientLoading(false);
        }
    };

    useEffect(() => {
        if (!loading) {
            loadFilterClientPage("", 0, true);
        }
    }, [loading]);

    const handleFilterClientScrollToBottom = () => {
        if (!filterClientLoading && filterClientHasMore) {
            const nextOffset = filterClientOffset + 100;
            setFilterClientOffset(nextOffset);
            loadFilterClientPage(filterClientSearch, nextOffset, false);
        }
    };

    const handleFilterClientInputChange = (val, actionMeta) => {
        if (actionMeta.action === "input-change") {
            setFilterClientSearch(val);
            if (filterClientSearchTimeout.current) clearTimeout(filterClientSearchTimeout.current);
            filterClientSearchTimeout.current = setTimeout(() => {
                setFilterClientOffset(0);
                loadFilterClientPage(val, 0, true);
            }, 300);
        }
    };

    // ── Pagination State for Categories Filter ────────────────────────────────
    const [filterCatOptions, setFilterCatOptions] = useState([]);
    const [filterCatLoading, setFilterCatLoading] = useState(false);
    const [filterCatSearch, setFilterCatSearch] = useState("");
    const [filterCatOffset, setFilterCatOffset] = useState(0);
    const [filterCatHasMore, setFilterCatHasMore] = useState(true);
    const filterCatSearchTimeout = useRef(null);

    const loadFilterCategoryPage = async (query = "", offset = 0, isNewSearch = false) => {
        setFilterCatLoading(true);
        try {
            const results = await handleSearchCategories(query, offset);
            const formatted = results.map(c => ({ value: c.id, label: c.name }));
            const clearOpt = { value: "", label: `Todas as ${config.labels.categories?.toLowerCase() || "categorias"}` };

            if (isNewSearch) {
                setFilterCatOptions([clearOpt, ...formatted]);
            } else {
                setFilterCatOptions(prev => {
                    const existingIds = new Set(prev.map(p => p.value));
                    const newItems = formatted.filter(f => !existingIds.has(f.value));
                    return [...prev, ...newItems];
                });
            }
            setFilterCatHasMore(results.length > 0 && results.length % 100 === 0);
        } catch (e) {
            console.error(e);
        } finally {
            setFilterCatLoading(false);
        }
    };

    useEffect(() => {
        if (!loading) {
            loadFilterCategoryPage("", 0, true);
        }
    }, [loading]);

    const handleFilterCatScrollToBottom = () => {
        if (!filterCatLoading && filterCatHasMore) {
            const nextOffset = filterCatOffset + 100;
            setFilterCatOffset(nextOffset);
            loadFilterCategoryPage(filterCatSearch, nextOffset, false);
        }
    };

    const handleFilterCatInputChange = (val, actionMeta) => {
        if (actionMeta.action === "input-change") {
            setFilterCatSearch(val);
            if (filterCatSearchTimeout.current) clearTimeout(filterCatSearchTimeout.current);
            filterCatSearchTimeout.current = setTimeout(() => {
                setFilterCatOffset(0);
                loadFilterCategoryPage(val, 0, true);
            }, 300);
        }
    };
    const { config, getGenderHelpers } = useConfig();
    const g = getGenderHelpers("contract");

    const filtersContainerRef = useRef(null);

    // Load initial data from cache (forceRefresh=false) — data is refreshed
    // after mutations (create/update/delete) via loadContracts(true).
    // StrictMode-safe guard: prevent double-firing of initial load
    const contractsInitLoadDone = useRef(false);
    useEffect(() => {
        if (contractsInitLoadDone.current) return;
        contractsInitLoadDone.current = true;
        loadInitialData(false);
    }, []);

    // Equalize filter button widths
    useEffect(() => {
        if (filtersContainerRef.current) {
            const buttons = filtersContainerRef.current.querySelectorAll(
                ".contracts-filter-button",
            );
            if (buttons.length > 0) {
                // Reset min-width to measure natural width
                buttons.forEach((btn) => (btn.style.minWidth = "auto"));

                // Calculate minimum button width
                let minButtonWidth = 0;
                buttons.forEach((btn) => {
                    const width = btn.offsetWidth;
                    if (width > minButtonWidth) {
                        minButtonWidth = width;
                    }
                });

                // Ensure minimum width of 120px
                minButtonWidth = Math.max(minButtonWidth, 120);

                // Apply minimum width to all buttons
                buttons.forEach((btn) => {
                    btn.style.minWidth = minButtonWidth + "px";
                });
            }
        }
    }, [filter, contracts]); // Re-calculate when filter or contracts change

    const loadInitialData = async (forceRefresh = false) => {
        setLoading(true);
        try {
            // Load enriched contracts first — they contain client_name, category_name,
            // subcategory_name inline, so the table renders immediately without
            // waiting for separate /clients and /categories calls.
            await loadContracts(forceRefresh);
        } finally {
            setLoading(false);
        }

        // Load auxiliary data in background (needed for filter dropdowns & modals only)
        Promise.all([
            loadClients(forceRefresh),
            loadCategories(forceRefresh),
        ]).catch((err) => console.warn("Background data load:", err));
    };

    const loadContracts = async (forceRefresh = false) => {
        setError("");
        try {
            const response = await fetchContracts({}, forceRefresh);
            setContracts(response.data || []);
        } catch (err) {
            setError(err.message);
        }
    };

    const loadClients = async (forceRefresh = false) => {
        try {
            const response = await fetchClients({}, forceRefresh);
            setClients(response.data || []);
        } catch (err) {
            console.error("Erro ao carregar clientes:", err);
        }
    };

    const loadCategories = async (forceRefresh = false) => {
        try {
            const response = await fetchCategories(forceRefresh);
            setCategories(response.data || []);
        } catch (err) {
            console.error("Erro ao carregar categorias:", err);
        }
    };

    const loadSubcategories = async (categoryId, forceRefresh = false) => {
        try {
            const response = await fetchSubcategories(categoryId, forceRefresh);
            setLines(response.data || []);
        } catch (err) {
            console.error("Erro ao carregar linhas:", err);
        }
    };

    const loadAffiliates = async (clientId) => {
        try {
            const data = await contractsApi.loadAffiliates(
                apiUrl,
                token,
                clientId,
                onTokenExpired,
            );
            setAffiliates(data);
        } catch (err) {
            console.error("Erro ao carregar afiliados:", err);
        }
    };

    const handleCreateContract = async () => {
        try {
            const apiData = prepareContractDataForAPI(formData);
            const result = await contractsApi.createContract(
                apiUrl,
                token,
                apiData,
                onTokenExpired,
            );
            const createdContractId = result?.data?.id;

            // Create financial if financial data exists
            if (
                createdContractId &&
                financialData &&
                financialData.financial_type
            ) {
                try {
                    await financialApi.createFinancial(
                        apiUrl,
                        token,
                        {
                            contract_id: createdContractId,
                            ...financialData,
                        },
                        onTokenExpired,
                    );
                } catch (financialErr) {
                    console.error("Error creating financial:", financialErr);
                    // Don't fail the contract creation if financial fails
                }
            }

            invalidateCache("contracts");
            await loadContracts(true);
            closeModal();
        } catch (err) {
            setError(err.message);
        }
    };

    const handleUpdateContract = async () => {
        setError("");
        try {
            const apiData = await contractsApi.updateContract(
                apiUrl,
                token,
                selectedContract.id,
                prepareContractDataForAPI(formData),
                onTokenExpired,
            );
            invalidateCache("contracts");
            await loadContracts(true);
            closeModal();
        } catch (err) {
            setError(err.message);
        }
    };

    const handleArchiveContract = async (contractId) => {
        if (!window.confirm("Tem certeza que deseja arquivar este contrato?"))
            return;

        try {
            await contractsApi.archiveContract(
                apiUrl,
                token,
                contractId,
                onTokenExpired,
            );
            invalidateCache("contracts");
            await loadContracts(true);
        } catch (err) {
            setError(err.message);
        }
    };

    const handleUnarchiveContract = async (contractId) => {
        try {
            await contractsApi.unarchiveContract(
                apiUrl,
                token,
                contractId,
                onTokenExpired,
            );
            invalidateCache("contracts");
            await loadContracts(true);
        } catch (err) {
            setError(err.message);
        }
    };

    const [deleteDialogContract, setDeleteDialogContract] = useState(null);

    const handleDeleteClick = (contract) => {
        const isArchived = !!contract.archived_at;
        const statusObj = getContractStatus(contract);

        // Se já está arquivado ou expirado, ou se não iniciou ainda, confirmar via window.confirm
        if (isArchived || statusObj.status === "Expirado" || statusObj.status === "Não Iniciado") {
            if (window.confirm("Deseja realmente excluir este contrato permanentemente? Esta ação não pode ser desfeita.")) {
                handleConfirmDelete(contract.id);
            }
        } else {
            // Contrato ativo / próximo ao vencimento: mostra o modal inteligente
            setDeleteDialogContract(contract);
        }
    };

    const handleConfirmDelete = async (contractId) => {
        setDeleteDialogContract(null);
        try {
            await contractsApi.deleteContract(
                apiUrl,
                token,
                contractId,
                onTokenExpired,
            );
            invalidateCache("contracts");
            await loadContracts(true);
        } catch (err) {
            setError(err.message);
        }
    };

    // Search clients for AsyncSelect in the modal
    const handleSearchClients = async (query, offset = 0) => {
        return contractsApi.searchClients(apiUrl, token, query, onTokenExpired, offset);
    };

    // Search categories for AsyncSelect in the modal and filters
    const handleSearchCategories = async (query, offset = 0) => {
        return contractsApi.searchCategories(apiUrl, token, query, onTokenExpired, offset);
    };

    const openCreateModal = () => {
        setModalMode("create");
        setFormData(getInitialFormData());
        setLines([]);
        setAffiliates([]);
        setFinancialData(null);
        setExistingFinancial(null);
        setShowModal(true);
    };

    const openEditModal = async (contract) => {
        setModalMode("edit");
        setSelectedContract(contract);
        setFormData(formatContractForEdit(contract));
        setShowModal(true);

        // Fetch fresh data in the background so fields are always up-to-date
        contractsApi.getContractByID(apiUrl, token, contract.id, onTokenExpired)
            .then((fresh) => {
                if (fresh) {
                    setSelectedContract(fresh);
                    setFormData(formatContractForEdit(fresh));
                }
            })
            .catch(() => { }); // fallback to already-set data

        if (contract.client_id) {
            loadAffiliates(contract.client_id);
        }
        if (contract.line?.category_id) {
            loadSubcategories(contract.line.category_id);
        }

        // Load existing financial for this contract
        try {
            const financial = await financialApi.getContractFinancial(
                apiUrl,
                token,
                contract.id,
                onTokenExpired,
            );
            setExistingFinancial(financial);
            setFinancialData(financial);
        } catch (err) {
            setExistingFinancial(null);
            setFinancialData(null);
        }
    };

    const openDetailsModal = (contract) => {
        setSelectedContract(contract);
        setShowDetailsModal(true);
    };

    const closeModal = () => {
        setShowModal(false);
        setSelectedContract(null);
        setFormData(getInitialFormData());
        setLines([]);
        setAffiliates([]);
        setFinancialData(null);
        setExistingFinancial(null);
        setError("");
    };

    const closeDetailsModal = () => {
        setShowDetailsModal(false);
        setSelectedContract(null);
    };

    const handleSubmit = (e) => {
        e.preventDefault();
        if (modalMode === "create") {
            handleCreateContract();
        } else {
            handleUpdateContract();
        }
    };

    const handleCategoryChange = (categoryId) => {
        if (categoryId) {
            loadSubcategories(categoryId);
        } else {
            setLines([]);
        }
    };

    const handleClientChange = (clientId) => {
        if (clientId) {
            loadAffiliates(clientId);
        } else {
            setAffiliates([]);
        }
    };

    function compareContracts(a, b) {
        const now = new Date();
        const aEnd = new Date(a.end_date);
        const bEnd = new Date(b.end_date);

        // Contratos arquivados sempre por último
        if (a.archived_at && !b.archived_at) return 1;
        if (!a.archived_at && b.archived_at) return -1;
        if (a.archived_at && b.archived_at) return 0;

        const aDiff = aEnd - now;
        const bDiff = bEnd - now;

        // Quanto mais negativo, mais em cima
        return aDiff - bDiff;
    }

    function sortContracts(contracts, sortBy, sortOrder, lookupMaps) {
        const clientById = lookupMaps?.clientById;
        const categoryBySubcategoryId = lookupMaps?.categoryBySubcategoryId;
        const subcategoryNameById = lookupMaps?.subcategoryNameById;

        return [...contracts].sort((a, b) => {
            let aVal, bVal;

            switch (sortBy) {
                case "model":
                    aVal = (a.model || "").toLowerCase();
                    bVal = (b.model || "").toLowerCase();
                    break;
                case "start_date":
                    // Contratos sem data de início vão para o final
                    if (!a.start_date && !b.start_date) return 0;
                    if (!a.start_date) return sortOrder === "asc" ? 1 : -1;
                    if (!b.start_date) return sortOrder === "asc" ? -1 : 1;
                    aVal = new Date(a.start_date);
                    bVal = new Date(b.start_date);
                    break;
                case "end_date":
                    // Contratos sem vencimento vão para o final em crescente, início em decrescente
                    if (!a.end_date && !b.end_date) return 0;
                    if (!a.end_date) return sortOrder === "asc" ? 1 : -1;
                    if (!b.end_date) return sortOrder === "asc" ? -1 : 1;
                    aVal = new Date(a.end_date);
                    bVal = new Date(b.end_date);
                    break;
                case "client":
                    // Use enriched field first, fallback to map lookup
                    aVal = (
                        a.client_name ||
                        clientById?.get(a.client_id)?.name ||
                        ""
                    ).toLowerCase();
                    bVal = (
                        b.client_name ||
                        clientById?.get(b.client_id)?.name ||
                        ""
                    ).toLowerCase();
                    break;
                case "category":
                    // Use enriched field first, fallback to map lookup
                    aVal = (
                        a.category_name ||
                        categoryBySubcategoryId?.get(a.subcategory_id)?.name ||
                        ""
                    ).toLowerCase();
                    bVal = (
                        b.category_name ||
                        categoryBySubcategoryId?.get(b.subcategory_id)?.name ||
                        ""
                    ).toLowerCase();
                    break;
                case "subcategory":
                    // Use enriched field first, fallback to map lookup
                    aVal = (
                        a.subcategory_name ||
                        subcategoryNameById?.get(a.subcategory_id) ||
                        ""
                    ).toLowerCase();
                    bVal = (
                        b.subcategory_name ||
                        subcategoryNameById?.get(b.subcategory_id) ||
                        ""
                    ).toLowerCase();
                    break;
                case "status":
                    // Simple status comparison
                    const getStatusOrder = (contract) => {
                        if (contract.archived_at) return 4;
                        const endDate = new Date(contract.end_date);
                        const now = new Date();
                        if (endDate < now) return 3; // expired
                        if (endDate - now < 30 * 24 * 60 * 60 * 1000) return 2; // expiring
                        return 1; // active
                    };
                    aVal = getStatusOrder(a);
                    bVal = getStatusOrder(b);
                    break;
                default:
                    return 0;
            }

            if (aVal < bVal) return sortOrder === "asc" ? -1 : 1;
            if (aVal > bVal) return sortOrder === "asc" ? 1 : -1;
            return 0;
        });
    }

    const clientById = useMemo(
        () => new Map(clients.map((client) => [client.id, client])),
        [clients],
    );

    const categoryBySubcategoryId = useMemo(() => {
        const map = new Map();
        for (const category of categories) {
            if (!Array.isArray(category.lines)) continue;
            for (const line of category.lines) {
                map.set(line.id, category);
            }
        }
        return map;
    }, [categories]);

    const subcategoryNameById = useMemo(() => {
        const map = new Map();
        for (const category of categories) {
            if (!Array.isArray(category.lines)) continue;
            for (const line of category.lines) {
                map.set(line.id, line.name);
            }
        }
        return map;
    }, [categories]);

    const filteredContractsAll = useMemo(
        () =>
            filterContracts(
                [...contracts].sort(compareContracts),
                filter,
                {
                    categoryId: categoryIdFilter,
                    subcategoryId: subcategoryIdFilter,
                    clientName: clientNameFilter,
                    contractName: contractNameFilter,
                },
                clients,
                categories,
                { clientById, categoryBySubcategoryId },
            ),
        [
            contracts,
            filter,
            categoryIdFilter,
            subcategoryIdFilter,
            clientNameFilter,
            contractNameFilter,
            clients,
            categories,
            clientById,
            categoryBySubcategoryId,
        ],
    );

    const allFilteredContracts = useMemo(
        () =>
            sortContracts(filteredContractsAll, sortBy, sortOrder, {
                clientById,
                categoryBySubcategoryId,
                subcategoryNameById,
            }),
        [
            filteredContractsAll,
            sortBy,
            sortOrder,
            clientById,
            categoryBySubcategoryId,
            subcategoryNameById,
        ],
    );

    // Pagination
    const totalItems = allFilteredContracts.length;
    const startIndex = (currentPage - 1) * itemsPerPage;
    const endIndex = startIndex + itemsPerPage;
    const filteredContracts = allFilteredContracts.slice(startIndex, endIndex);

    return (
        <div className="contracts-container">
            <div className="contracts-header">
                <h1 className="contracts-title">
                    📄 {config.labels.contracts || "Contratos"}
                </h1>
                <div className="button-group">
                    <PrimaryButton onClick={openCreateModal}>
                        + {g.new} {config.labels.contract}
                    </PrimaryButton>
                </div>
            </div>

            {error && <div className="contracts-error">{error}</div>}

            <div className="contracts-filters" ref={filtersContainerRef}>
                <button
                    onClick={() => setFilter("all")}
                    className={`contracts-filter-button ${filter === "all" ? "active-all" : ""}`}
                >
                    {g.all} ({contracts.filter((c) => !c.archived_at).length})
                </button>
                <button
                    onClick={() => setFilter("active")}
                    className={`contracts-filter-button ${filter === "active" ? "active-active" : ""}`}
                >
                    {g.active}
                </button>
                <button
                    onClick={() => setFilter("not-started")}
                    className={`contracts-filter-button ${filter === "not-started" ? "active-not-started" : ""}`}
                >
                    Não Iniciados
                </button>
                <button
                    onClick={() => setFilter("expiring")}
                    className={`contracts-filter-button ${filter === "expiring" ? "active-expiring" : ""}`}
                >
                    Expirando
                </button>
                <button
                    onClick={() => setFilter("expired")}
                    className={`contracts-filter-button ${filter === "expired" ? "active-expired" : ""}`}
                >
                    Expirados
                </button>
                <button
                    onClick={() => setFilter("archived")}
                    className={`contracts-filter-button ${filter === "archived" ? "active-archived" : ""}`}
                >
                    {config.labels.archived || g.archived}
                </button>
            </div>

            <div className="contracts-search-filters">
                <Select
                    options={filterClientOptions}
                    onInputChange={handleFilterClientInputChange}
                    onMenuScrollToBottom={handleFilterClientScrollToBottom}
                    value={
                        clientNameFilter
                            ? { value: clientNameFilter, label: clientNameFilter }
                            : null
                    }
                    onChange={(selected) => setClientNameFilter(selected ? selected.value : "")}
                    isClearable
                    isSearchable
                    filterOption={null}
                    placeholder={`Filtrar por ${config.labels.client?.toLowerCase() || "cliente"}...`}
                    styles={customSelectStyles}
                />

                <Select
                    options={filterCatOptions}
                    onInputChange={handleFilterCatInputChange}
                    onMenuScrollToBottom={handleFilterCatScrollToBottom}
                    value={
                        categoryIdFilter
                            ? {
                                value: categoryIdFilter,
                                label:
                                    categories.find(
                                        (c) => c.id === categoryIdFilter,
                                    )?.name || "Selecionado",
                            }
                            : null
                    }
                    onChange={(selected) => setCategoryIdFilter(selected ? selected.value : "")}
                    isClearable
                    isSearchable
                    filterOption={null}
                    placeholder={`Selecionar ${config.labels.categories?.toLowerCase() || "categoria"}...`}
                    styles={customSelectStyles}
                />

                <Select
                    value={
                        subcategoryIdFilter
                            ? {
                                value: subcategoryIdFilter,
                                label:
                                    availableSubcategories.find(
                                        (s) => s.id === subcategoryIdFilter,
                                    )?.name || "",
                            }
                            : null
                    }
                    onChange={(selected) =>
                        setSubcategoryIdFilter(selected ? selected.value : "")
                    }
                    options={[
                        {
                            value: "",
                            label: `Todas as ${config.labels.subcategories?.toLowerCase() || "subcategorias"}`,
                        },
                        ...availableSubcategories
                            .filter((s) => !s.archived_at)
                            .map((sub) => ({
                                value: sub.id,
                                label: sub.name,
                            })),
                    ]}
                    isSearchable={true}
                    isDisabled={!categoryIdFilter}
                    placeholder={`Selecionar ${config.labels.subcategories?.toLowerCase() || "subcategoria"}`}
                    styles={customSelectStyles}
                />

                <div
                    style={{
                        display: "flex",
                        alignItems: "center",
                        gap: "8px",
                    }}
                >
                    <span
                        style={{
                            fontSize: "14px",
                            fontWeight: "500",
                            color: "#2c3e50",
                            whiteSpace: "nowrap",
                        }}
                    >
                        Ordenar por:
                    </span>
                    <Select
                        value={
                            sortBy
                                ? {
                                    value: sortBy,
                                    label: (() => {
                                        const sortLabels = {
                                            model:
                                                config.labels.model ||
                                                "Nome/Modelo",
                                            start_date: "Data de Início",
                                            end_date: "Data de Vencimento",
                                            client:
                                                config.labels.client ||
                                                "Cliente",
                                            status: "Status",
                                        };
                                        return sortLabels[sortBy];
                                    })(),
                                }
                                : null
                        }
                        onChange={(selected) => {
                            if (selected) {
                                setSortBy(selected.value);
                            }
                        }}
                        options={[
                            {
                                value: "model",
                                label: config.labels.model || "Nome/Modelo",
                            },
                            {
                                value: "start_date",
                                label: "Data de Início",
                            },
                            {
                                value: "end_date",
                                label: "Data de Vencimento",
                            },
                            {
                                value: "client",
                                label: config.labels.client || "Cliente",
                            },
                            {
                                value: "category",
                                label: config.labels.category || "Categoria",
                            },
                            {
                                value: "subcategory",
                                label:
                                    config.labels.subcategory || "Subcategoria",
                            },
                            { value: "status", label: "Status" },
                        ]}
                        isSearchable={false}
                        placeholder="Selecionar..."
                        styles={customSelectStyles}
                    />

                    <Select
                        value={
                            sortOrder
                                ? {
                                    value: sortOrder,
                                    label:
                                        sortOrder === "asc"
                                            ? "Crescente"
                                            : "Decrescente",
                                }
                                : null
                        }
                        onChange={(selected) => {
                            if (selected) {
                                setSortOrder(selected.value);
                            }
                        }}
                        options={[
                            { value: "asc", label: "Crescente" },
                            { value: "desc", label: "Decrescente" },
                        ]}
                        isSearchable={false}
                        placeholder="Ordem..."
                        styles={customSelectStyles}
                    />
                </div>

                <input
                    type="text"
                    placeholder={`Filtrar por ${config.labels.model?.toLowerCase() || "descrição"}...`}
                    value={contractNameFilter}
                    onChange={(e) => setContractNameFilter(e.target.value)}
                    className="contracts-search-input"
                    style={{
                        minWidth: "220px",
                        width: "220px",
                        fontSize: "14px",
                        padding: "8px 12px",
                        border: "1px solid var(--border-color, #ddd)",
                        borderRadius: "4px",
                        background: "var(--content-bg, white)",
                        color: "var(--primary-text-color, #333)",
                    }}
                />

                <PrimaryButton
                    onClick={() => {
                        updateValuesImmediate({
                            categoryId: "",
                            subcategoryId: "",
                            clientName: "",
                            contractName: "",
                            page: "1",
                        });
                    }}
                    title="Limpar filtros"
                    style={{
                        minWidth: "120px",
                    }}
                >
                    Limpar Filtros
                </PrimaryButton>
            </div>

            <div className="contracts-table-wrapper">
                {/* <div className="contracts-table-header">
                    <h2 className="contracts-table-header-title">Contratos</h2>
                </div>*/}
                <ContractsTable
                    filteredContracts={filteredContracts}
                    clients={clients}
                    categories={categories}
                    onEdit={openEditModal}
                    onArchive={handleArchiveContract}
                    onUnarchive={handleUnarchiveContract}
                    onDelete={handleDeleteClick}
                />
            </div>

            <Pagination
                currentPage={currentPage}
                totalItems={totalItems}
                itemsPerPage={itemsPerPage}
                onPageChange={setCurrentPage}
                onItemsPerPageChange={setItemsPerPage}
            />

            <ContractModal
                showModal={showModal}
                modalMode={modalMode}
                formData={formData}
                setFormData={setFormData}
                clients={clients}
                categories={categories}
                lines={lines}
                affiliates={affiliates}
                onSubmit={handleSubmit}
                onClose={closeModal}
                onCategoryChange={handleCategoryChange}
                onClientChange={handleClientChange}
                onSearchClients={handleSearchClients}
                onSearchCategories={handleSearchCategories}
                error={error}
                financialData={existingFinancial}
                onFinancialChange={setFinancialData}
                showFinancialSection={true}
                showFinancialValues={true}
                canEditFinancialValues={true}
            />

            {/* Smart Delete Dialog */}
            {deleteDialogContract && (
                <DeleteContractDialog
                    contractId={deleteDialogContract.id}
                    onArchive={handleArchiveContract}
                    onDelete={handleConfirmDelete}
                    onClose={() => setDeleteDialogContract(null)}
                />
            )}

            {showDetailsModal && selectedContract && (
                <div
                    style={{
                        position: "fixed",
                        top: 0,
                        left: 0,
                        right: 0,
                        bottom: 0,
                        background: "rgba(0,0,0,0.5)",
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "center",
                        zIndex: 1000,
                        padding: "20px",
                    }}
                    onClick={closeDetailsModal}
                >
                    <div
                        onClick={(e) => e.stopPropagation()}
                        style={{
                            background: "white",
                            borderRadius: "8px",
                            padding: "32px",
                            width: "90%",
                            maxWidth: "600px",
                            maxHeight: "90vh",
                            overflowY: "auto",
                        }}
                    >
                        <h2
                            style={{
                                marginTop: 0,
                                marginBottom: "24px",
                                fontSize: "24px",
                                color: "#2c3e50",
                            }}
                        >
                            Detalhes do {config.labels.contract || "Contrato"}
                        </h2>

                        <div style={{ display: "grid", gap: "16px" }}>
                            <DetailRow
                                label={config.labels.model || "Descrição"}
                                value={selectedContract.model || "-"}
                            />
                            <DetailRow
                                label={
                                    config.labels.item_key || "Identificador"
                                }
                                value={selectedContract.item_key || "-"}
                            />
                            <DetailRow
                                label={config.labels.client || "Cliente"}
                                value={getClientName(
                                    selectedContract.client_id,
                                    clients,
                                )}
                            />
                            {selectedContract.affiliate && (
                                <DetailRow
                                    label={
                                        config.labels.affiliate || "Afiliado"
                                    }
                                    value={selectedContract.affiliate.name}
                                />
                            )}
                            <DetailRow
                                label={config.labels.category || "Categoria"}
                                value={
                                    selectedContract.subcategory_id
                                        ? categoryBySubcategoryId.get(
                                            selectedContract.subcategory_id,
                                        )?.name || "-"
                                        : "-"
                                }
                            />
                            <DetailRow
                                label={
                                    config.labels.subcategory || "Subcategoria"
                                }
                                value={
                                    selectedContract.subcategory_id
                                        ? subcategoryNameById.get(
                                            selectedContract.subcategory_id,
                                        ) ||
                                        selectedContract.line?.name ||
                                        "-"
                                        : "-"
                                }
                            />
                            <DetailRow
                                label="Data de Início"
                                value={formatDate(selectedContract.start_date)}
                            />
                            <DetailRow
                                label="Data de Vencimento"
                                value={formatDate(selectedContract.end_date)}
                            />
                        </div>

                        <div
                            style={{
                                marginTop: "32px",
                                display: "flex",
                                justifyContent: "flex-end",
                            }}
                        >
                            <button
                                onClick={closeDetailsModal}
                                style={{
                                    padding: "10px 24px",
                                    background: "#3498db",
                                    color: "white",
                                    border: "none",
                                    borderRadius: "4px",
                                    cursor: "pointer",
                                    fontSize: "14px",
                                }}
                            >
                                Fechar
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}

function DetailRow({ label, value }) {
    return (
        <div
            style={{
                padding: "12px",
                background: "#f8f9fa",
                borderRadius: "6px",
            }}
        >
            <div
                style={{
                    fontSize: "12px",
                    color: "#7f8c8d",
                    marginBottom: "4px",
                    fontWeight: "600",
                }}
            >
                {label}
            </div>
            <div
                style={{
                    fontSize: "14px",
                    color: "#2c3e50",
                }}
            >
                {value}
            </div>
        </div>
    );
}

/**
 * DeleteContractDialog - shown when deleting an Active contract.
 * Offers options to Archive instead, Cancel, or Delete Permanently.
 */
function DeleteContractDialog({ contractId, onArchive, onDelete, onClose }) {
    const [loading, setLoading] = React.useState(false);

    const handleArchive = async () => {
        setLoading(true);
        await onArchive(contractId);
        setLoading(false);
        onClose();
    };

    const handleDelete = async () => {
        setLoading(true);
        await onDelete(contractId);
        setLoading(false);
        onClose();
    };

    return (
        <div
            style={{
                position: "fixed",
                inset: 0,
                background: "rgba(0,0,0,0.55)",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                zIndex: 2000,
            }}
            onClick={onClose}
        >
            <div
                onClick={(e) => e.stopPropagation()}
                style={{
                    background: "var(--content-bg, white)",
                    borderRadius: "10px",
                    padding: "32px",
                    width: "420px",
                    boxShadow: "0 8px 32px rgba(0,0,0,0.18)",
                    textAlign: "center"
                }}
            >
                <h3 style={{ margin: "0 0 16px 0", fontSize: "18px", color: "#2c3e50" }}>
                    Tem certeza que deseja excluir?
                </h3>
                <p style={{ margin: "0 0 24px 0", fontSize: "14px", color: "#7f8c8d", lineHeight: "1.5" }}>
                    Este contrato ainda está ativo. Em vez de excluí-lo definitivamente (o que apagará todo o histórico), sugerimos arquivá-lo.
                </p>

                <div style={{ display: "flex", flexDirection: "column", gap: "10px" }}>
                    <button
                        onClick={handleArchive}
                        disabled={loading}
                        style={{
                            padding: "12px",
                            background: "#3498db",
                            color: "white",
                            border: "none",
                            borderRadius: "6px",
                            cursor: loading ? "wait" : "pointer",
                            fontSize: "14px",
                            fontWeight: "600",
                        }}
                    >
                        {loading ? "Processando..." : "Arquivar em vez de excluir"}
                    </button>

                    <button
                        onClick={onClose}
                        disabled={loading}
                        style={{
                            padding: "12px",
                            background: "#95a5a6",
                            color: "white",
                            border: "none",
                            borderRadius: "6px",
                            cursor: "pointer",
                            fontSize: "14px",
                            fontWeight: "600",
                        }}
                    >
                        Desistir
                    </button>

                    <button
                        onClick={handleDelete}
                        disabled={loading}
                        style={{
                            padding: "12px",
                            background: "transparent",
                            color: "#e74c3c",
                            border: "1px solid #e74c3c",
                            borderRadius: "6px",
                            cursor: loading ? "wait" : "pointer",
                            fontSize: "14px",
                            fontWeight: "600",
                            marginTop: "12px",
                        }}
                    >
                        {loading ? "Processando..." : "Excluir permanentemente"}
                    </button>
                </div>
            </div>
        </div>
    );
}
