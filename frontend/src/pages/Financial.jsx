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
import { useConfig } from "../contexts/ConfigContext";
import { useData } from "../contexts/DataContext";
import { financialApi } from "../api/financialApi";
import { useUrlState } from "../hooks/useUrlState";
import PrimaryButton from "../components/common/PrimaryButton";
import Pagination from "../components/common/Pagination";
import "./styles/Financial.css";

// Hook personalizado para preservar scroll
const useScrollPreservation = () => {
    const scrollRef = useRef(null);
    const scrollPositionRef = useRef(0);

    const saveScrollPosition = () => {
        if (scrollRef.current) {
            scrollPositionRef.current = scrollRef.current.scrollTop;
        }
    };

    const restoreScrollPosition = () => {
        if (scrollRef.current) {
            scrollRef.current.scrollTop = scrollPositionRef.current;
        }
    };

    return { scrollRef, saveScrollPosition, restoreScrollPosition };
};

export default function Financial({ token, apiUrl, onTokenExpired }) {
    const { fetchContracts, fetchClients, fetchCategories } = useData();
    const { config } = useConfig();
    const [financialRecords, setFinancialRecords] = useState([]);
    const [contracts, setContracts] = useState([]);
    const [clients, setClients] = useState([]);
    const [categories, setCategories] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");
    const [detailedSummary, setDetailedSummary] = useState(null);
    const [detailedLoading, setDetailedLoading] = useState(false);
    const [detailedLoaded, setDetailedLoaded] = useState(false);
    const [upcomingFinancials, setUpcomingFinancials] = useState([]);
    const [overdueFinancials, setOverdueFinancials] = useState([]);
    const [upcomingCurrentPage, setUpcomingCurrentPage] = useState(1);
    const [upcomingItemsPerPage, setUpcomingItemsPerPage] = useState(10);
    const [overdueCurrentPage, setOverdueCurrentPage] = useState(1);
    const [overdueItemsPerPage, setOverdueItemsPerPage] = useState(10);
    const filtersContainerRef = useRef(null);

    // URL state for filters
    const { values, updateValue, updateValues, updateValuesImmediate } =
        useUrlState(
            {
                filter: "all",
                search: "",
                page: "1",
                limit: "20",
                financialType: "",
                status: "",
                periodView: "current",
                mainTab: "dashboard", // dashboard, table
                clientFilter: "",
                contractFilter: "",
                categoryFilter: "",
                subcategoryFilter: "",
                descriptionFilter: "",
                sortBy: "",
                sortOrder: "asc",
                showCompleted: "false",
                startDate: "",
                endDate: "",
            },
            { debounce: true, debounceTime: 300, syncWithUrl: false },
        );

    const filter = values.filter;
    const searchTerm = values.search;
    const currentPage = parseInt(values.page || "1", 10);
    const itemsPerPage = parseInt(values.limit || "20", 10);
    const financialTypeFilter = values.financialType;
    const statusFilter = values.status;
    const periodView = values.periodView;
    const mainTab = values.mainTab;
    const clientFilter = values.clientFilter;
    const contractFilter = values.contractFilter;
    const categoryFilter = values.categoryFilter;
    const subcategoryFilter = values.subcategoryFilter;
    const descriptionFilter = values.descriptionFilter;
    const sortBy = values.sortBy;
    const sortOrder = values.sortOrder;
    const showCompleted = values.showCompleted === "true";

    const setFilter = (val) =>
        updateValuesImmediate({ filter: val, page: "1" });
    const setSearchTerm = (val) => updateValues({ search: val, page: "1" });
    const setCurrentPage = (page) =>
        updateValuesImmediate({ page: page.toString() });
    const setItemsPerPage = (limit) =>
        updateValuesImmediate({ limit: limit.toString(), page: "1" });
    const setFinancialTypeFilter = (val) =>
        updateValuesImmediate({ financialType: val, page: "1" });
    const setStatusFilter = (val) =>
        updateValuesImmediate({ status: val, page: "1" });
    const setPeriodView = (val) => updateValuesImmediate({ periodView: val });
    const setMainTab = (val) => updateValuesImmediate({ mainTab: val });
    const setClientFilter = (val) =>
        updateValuesImmediate({ clientFilter: val, page: "1" });
    const setContractFilter = (val) =>
        updateValuesImmediate({ contractFilter: val, page: "1" });
    const setCategoryFilter = (val) =>
        updateValuesImmediate({ categoryFilter: val, page: "1" });
    const setSubcategoryFilter = (val) =>
        updateValuesImmediate({ subcategoryFilter: val, page: "1" });
    const setDescriptionFilter = (val) =>
        updateValuesImmediate({ descriptionFilter: val, page: "1" });
    const setSortBy = (val) =>
        updateValuesImmediate({ sortBy: val, page: "1" });
    const setSortOrder = (val) =>
        updateValuesImmediate({ sortOrder: val, page: "1" });
    const setStartDate = (val) =>
        updateValuesImmediate({ startDate: val, page: "1" });
    const setEndDate = (val) =>
        updateValuesImmediate({ endDate: val, page: "1" });
    const setShowCompleted = (val) =>
        updateValuesImmediate({ showCompleted: val.toString(), page: "1" });

    const clearFilters = () => {
        updateValuesImmediate({
            financialType: "",
            status: "",
            clientFilter: "",
            contractFilter: "",
            categoryFilter: "",
            subcategoryFilter: "",
            descriptionFilter: "",
            sortBy: "",
            sortOrder: "asc",
            showCompleted: "false",
            startDate: "",
            endDate: "",
            page: "1",
        });
    };

    const contractsById = useMemo(
        () => new Map(contracts.map((contract) => [contract.id, contract])),
        [contracts],
    );
    const clientsById = useMemo(
        () => new Map(clients.map((client) => [client.id, client])),
        [clients],
    );
    const categoriesById = useMemo(
        () => new Map(categories.map((category) => [category.id, category])),
        [categories],
    );

    // Get subcategories from selected category
    const availableSubcategories = categoryFilter
        ? categoriesById.get(categoryFilter)?.lines || []
        : categories.flatMap((c) => c.lines || []);

    // Modal states
    const [showModal, setShowModal] = useState(false);
    const [modalMode, setModalMode] = useState("view"); // 'view', 'edit'
    const [selectedFinancial, setSelectedFinancial] = useState(null);
    const [selectedInstallments, setSelectedInstallments] = useState([]);

    // Active tab for dashboard widgets - default to overdue
    const [activeTab, setActiveTab] = useState("overdue");

    // Active tab for period view
    const [periodTab, setPeriodTab] = useState("monthlyExpanded");

    // Custom period configuration
    const [customPeriodType, setCustomPeriodType] = useState("mensal"); // "mensal" or "anual"
    const [customStartDate, setCustomStartDate] = useState("");
    const [customEndDate, setCustomEndDate] = useState("");

    // Hook para preservar scroll do modal
    const { scrollRef, saveScrollPosition, restoreScrollPosition } =
        useScrollPreservation();

    // StrictMode-safe guard: prevent double-firing of initial load
    const initialLoadDone = useRef(false);
    useEffect(() => {
        if (initialLoadDone.current) return;
        initialLoadDone.current = true;
        loadData();
    }, []);

    // Lazy-load detailedSummary only when user switches to the dashboard tab
    // StrictMode-safe guard
    const detailedSummaryLoadStarted = useRef(false);
    useEffect(() => {
        if (mainTab === "dashboard" && !detailedLoaded && !detailedLoading && !detailedSummaryLoadStarted.current) {
            detailedSummaryLoadStarted.current = true;
            loadDetailedSummary();
        }
    }, [mainTab, detailedLoaded, detailedLoading]);

    // Prevent body scroll when modal is open
    useEffect(() => {
        if (showModal) {
            document.body.classList.add("modal-open");
        } else {
            document.body.classList.remove("modal-open");
        }

        // Cleanup on unmount
        return () => {
            document.body.classList.remove("modal-open");
        };
    }, [showModal]);

    // Load only the lists data (upcoming and overdue) without full page reload
    const loadListsData = async () => {
        try {
            const [upcomingData, overdueData] = await Promise.all([
                financialApi
                    .getUpcomingFinancial(apiUrl, token, 30, onTokenExpired)
                    .catch(() => []),
                financialApi
                    .getOverdueFinancial(apiUrl, token, onTokenExpired)
                    .catch(() => []),
            ]);

            setUpcomingFinancials(upcomingData || []);
            setOverdueFinancials(overdueData || []);
        } catch (err) {
            // Silently fail for list updates to avoid disrupting user experience
            console.warn("Failed to reload lists:", err);
        }
    };

    const loadData = async (silent = false) => {
        if (!silent) setLoading(true);
        setError("");

        try {
            // Primary load: enriched financial records contain contract_model,
            // client_name, category_name, subcategory_name inline — so the table
            // renders immediately without waiting for /contracts, /clients, /categories.
            const financialRecordsData = await financialApi.loadFinancial(
                apiUrl,
                token,
                onTokenExpired,
            );

            const normalizedFinancialRecords = Array.isArray(
                financialRecordsData,
            )
                ? financialRecordsData
                : financialRecordsData?.data || [];
            setFinancialRecords(normalizedFinancialRecords);
        } catch (err) {
            setError(err.message);
        } finally {
            setLoading(false);
        }

        // Background load: auxiliary data for filter dropdowns, modals, and lists.
        // These don't block initial render. Use cache (forceRefresh=false) since
        // this data may already be loaded by Dashboard or Contracts pages.
        Promise.all([
            fetchContracts({}, false)
                .then((res) => setContracts(res?.data || []))
                .catch(() => { }),
            fetchClients({}, false)
                .then((res) => setClients(res?.data || []))
                .catch(() => { }),
            fetchCategories(false)
                .then((res) => setCategories(res?.data || []))
                .catch(() => { }),
            financialApi
                .getUpcomingFinancial(apiUrl, token, 30, onTokenExpired)
                .then((data) => setUpcomingFinancials(data || []))
                .catch(() => { }),
            financialApi
                .getOverdueFinancial(apiUrl, token, onTokenExpired)
                .then((data) => setOverdueFinancials(data || []))
                .catch(() => { }),
        ]).catch((err) => console.warn("Background data load:", err));
    };

    // Lazy-load detailed summary — only called when user opens the dashboard tab
    const loadDetailedSummary = async () => {
        setDetailedLoading(true);
        try {
            const data = await financialApi
                .getDetailedSummary(apiUrl, token, onTokenExpired)
                .catch(() => null);
            setDetailedSummary(data);
            setDetailedLoaded(true);
        } catch (err) {
            console.warn("Failed to load detailed summary:", err);
        } finally {
            setDetailedLoading(false);
        }
    };

    // Get contract info for a financial
    const getContractInfo = (financial) => {
        // Use enriched fields from backend JOIN when available (instant, no map lookup)
        if (financial.contract_model || financial.client_name) {
            return {
                model: financial.contract_model || "—",
                clientName: financial.client_name || "—",
                clientNickname: financial.client_nickname,
            };
        }
        // Fallback to map lookup for backward compatibility
        const contract = contractsById.get(financial.contract_id);
        if (!contract) return { model: "—", clientName: "—" };

        const client = clientsById.get(contract.client_id);
        return {
            model: contract.model || "—",
            clientName: client?.name || "—",
            clientNickname: client?.nickname,
        };
    };

    // Filter financialRecords
    const filteredFinancial = financialRecords.filter((financial) => {
        const contractInfo = getContractInfo(financial);
        const contract = contractsById.get(financial.contract_id);

        // Date Range filter (using contract dates)
        if (values.startDate) {
            const startLimit = new Date(values.startDate);
            const contractStart =
                financial.contract_start || contract?.start_date
                    ? new Date(financial.contract_start || contract.start_date)
                    : null;
            if (contractStart && contractStart < startLimit) {
                // Se o contrato começou antes da data de início, talvez queiramos filtrar?
                // Ou talvez queiramos filtrar se ele está ATIVO no range.
            }
            // Simplificando: filtrar se o contrato termina antes do início do range ou começa depois do fim do range
        }

        const effectiveEndDate = financial.contract_end || contract?.end_date;
        const effectiveStartDate =
            financial.contract_start || contract?.start_date;
        if (values.startDate && effectiveEndDate) {
            if (new Date(effectiveEndDate) < new Date(values.startDate))
                return false;
        }
        if (values.endDate && effectiveStartDate) {
            if (new Date(effectiveStartDate) > new Date(values.endDate))
                return false;
        }

        // Financial type filter
        if (
            financialTypeFilter &&
            financial.financial_type !== financialTypeFilter
        ) {
            return false;
        }

        // Status filter (based on installments for personalizado)
        if (statusFilter) {
            if (financial.financial_type === "personalizado") {
                const hasPending = financial.pending_installments > 0;
                const allPaid =
                    financial.paid_installments ===
                    financial.total_installments;
                if (statusFilter === "pending" && !hasPending) return false;
                if (statusFilter === "paid" && !allPaid) return false;
            }
        }

        // Client filter
        if (clientFilter && contractInfo.clientName !== clientFilter) {
            return false;
        }

        // Contract filter
        if (contractFilter && contractInfo.model !== contractFilter) {
            return false;
        }

        // Category filter — use enriched category_name for fast match when possible
        if (categoryFilter) {
            if (financial.category_name) {
                // Enriched path: match by category ID via the categories map
                const category = categoriesById.get(categoryFilter);
                if (!category || category.name !== financial.category_name) {
                    return false;
                }
            } else if (contract) {
                const category = categoriesById.get(categoryFilter);
                const subcategory = category?.lines?.find(
                    (s) => s.id === contract.subcategory_id,
                );
                if (!subcategory) {
                    return false;
                }
            } else {
                return false;
            }
        }

        // Subcategory filter
        if (subcategoryFilter) {
            const contractSubcategoryId = contract?.subcategory_id;
            if (contractSubcategoryId !== subcategoryFilter) {
                return false;
            }
        }

        // Description filter
        if (descriptionFilter) {
            const desc = (financial.description || "").toLowerCase();
            if (!desc.includes(descriptionFilter.toLowerCase())) {
                return false;
            }
        }

        // Show completed filter
        if (!showCompleted && financial.financial_type === "personalizado") {
            const allPaid =
                financial.paid_installments === financial.total_installments;
            if (allPaid) return false;
        }

        return true;
    });

    // Sort filtered financial records
    const sortedFinancial = [...filteredFinancial].sort((a, b) => {
        if (!sortBy) return 0;

        const contractA = contractsById.get(a.contract_id);
        const contractB = contractsById.get(b.contract_id);

        let aVal, bVal;

        switch (sortBy) {
            case "client":
                // Use enriched field first, fallback to map lookup
                aVal = (
                    a.client_name ||
                    (contractA
                        ? clientsById.get(contractA.client_id)?.name
                        : null) ||
                    ""
                ).toLowerCase();
                bVal = (
                    b.client_name ||
                    (contractB
                        ? clientsById.get(contractB.client_id)?.name
                        : null) ||
                    ""
                ).toLowerCase();
                break;
            case "contract":
                // Use enriched field first, fallback to map lookup
                aVal = (
                    a.contract_model ||
                    contractA?.model ||
                    ""
                ).toLowerCase();
                bVal = (
                    b.contract_model ||
                    contractB?.model ||
                    ""
                ).toLowerCase();
                break;
            case "type":
                aVal = a.financial_type || "";
                bVal = b.financial_type || "";
                break;
            case "value":
                aVal = a.total_client_value || a.client_value || 0;
                bVal = b.total_client_value || b.client_value || 0;
                break;
            case "progress":
                aVal =
                    a.total_installments > 0
                        ? (a.paid_installments || 0) / a.total_installments
                        : 0;
                bVal =
                    b.total_installments > 0
                        ? (b.paid_installments || 0) / b.total_installments
                        : 0;
                break;
            default:
                return 0;
        }

        if (aVal < bVal) return sortOrder === "asc" ? -1 : 1;
        if (aVal > bVal) return sortOrder === "asc" ? 1 : -1;
        return 0;
    });

    // Pagination
    const totalItems = sortedFinancial.length;
    const startIndex = (currentPage - 1) * itemsPerPage;
    const endIndex = startIndex + itemsPerPage;
    const paginatedFinancial = sortedFinancial.slice(startIndex, endIndex);

    // Open financial details modal
    const openFinancialModal = async (financial, event) => {
        // Prevent scroll behavior
        if (event) {
            event.preventDefault();
            event.stopPropagation();
        }

        setSelectedFinancial(financial);
        setModalMode("view");

        // Load installments for any financial type
        try {
            const installments = await financialApi.getInstallments(
                apiUrl,
                token,
                financial.id,
                onTokenExpired,
            );
            setSelectedInstallments(installments || []);
        } catch (err) {
            setSelectedInstallments([]);
        }

        setShowModal(true);
    };

    const closeModal = (event) => {
        // Prevent scroll behavior
        if (event) {
            event.preventDefault();
            event.stopPropagation();
        }
        setShowModal(false);
        setSelectedFinancial(null);
        setSelectedInstallments([]);
    };

    // Mark installment as paid
    const handleMarkPaid = async (installmentId, financialId = null, event) => {
        // Prevent scroll behavior
        if (event) {
            event.preventDefault();
            event.stopPropagation();
        }

        // Save scroll position before API call
        saveScrollPosition();

        try {
            const pid = financialId || selectedFinancial?.id;
            if (!pid) return;

            await financialApi.markInstallmentPaid(
                apiUrl,
                token,
                pid,
                installmentId,
                onTokenExpired,
            );

            // Reload installments for the financial
            const installments = await financialApi.getInstallments(
                apiUrl,
                token,
                pid,
                onTokenExpired,
            );
            setSelectedInstallments(installments || []);

            // Update totals
            const paidCount = installments.filter(
                (inst) => inst.status === "pago",
            ).length;
            const totalCount = installments.length;

            // Update selectedFinancial if modal is open
            if (selectedFinancial && selectedFinancial.id === pid) {
                setSelectedFinancial((prev) =>
                    prev
                        ? {
                            ...prev,
                            paid_installments: paidCount,
                            total_installments: totalCount,
                        }
                        : null,
                );
            }

            // Update financialRecords list for table
            setFinancialRecords((prevFinancial) =>
                prevFinancial.map((p) =>
                    p.id === pid
                        ? {
                            ...p,
                            paid_installments: paidCount,
                            total_installments: totalCount,
                        }
                        : p,
                ),
            );

            // Update all data silently for automatic refresh
            loadData(true);
            // Also refresh detailed summary if it was loaded
            if (detailedLoaded) {
                loadDetailedSummary();
            }

            // Restore scroll position after state update
            setTimeout(restoreScrollPosition, 0);
        } catch (err) {
            setError(err.message);
        }
    };

    // Mark installment as pending (unpay)
    const handleMarkPending = async (installmentId, event) => {
        // Prevent scroll behavior
        if (event) {
            event.preventDefault();
            event.stopPropagation();
        }

        // Save scroll position before API call
        saveScrollPosition();

        try {
            await financialApi.markInstallmentPending(
                apiUrl,
                token,
                selectedFinancial.id,
                installmentId,
                onTokenExpired,
            );

            // Reload installments
            const installments = await financialApi.getInstallments(
                apiUrl,
                token,
                selectedFinancial.id,
                onTokenExpired,
            );
            setSelectedInstallments(installments || []);

            // Update totals
            const paidCount = installments.filter(
                (inst) => inst.status === "pago",
            ).length;
            const totalCount = installments.length;

            // Update selectedFinancial
            setSelectedFinancial((prev) =>
                prev
                    ? {
                        ...prev,
                        paid_installments: paidCount,
                        total_installments: totalCount,
                    }
                    : null,
            );

            // Update financialRecords list for table
            setFinancialRecords((prevFinancial) =>
                prevFinancial.map((p) =>
                    p.id === selectedFinancial.id
                        ? {
                            ...p,
                            paid_installments: paidCount,
                            total_installments: totalCount,
                        }
                        : p,
                ),
            );

            // Update all data silently for automatic refresh
            loadData(true);
            // Also refresh detailed summary if it was loaded
            if (detailedLoaded) {
                loadDetailedSummary();
            }

            // Restore scroll position after state update
            setTimeout(restoreScrollPosition, 0);
        } catch (err) {
            setError(err.message);
        }
    };

    // Custom select styles
    const customSelectStyles = {
        control: (provided, state) => ({
            ...provided,
            minWidth: "180px",
            fontSize: "14px",
            border: "1px solid var(--border-color, #ddd)",
            borderRadius: "4px",
            background: "var(--content-bg, white)",
            color: "var(--primary-text-color, #333)",
            cursor: "pointer",
            boxShadow: state.isFocused
                ? "0 0 0 1px var(--primary-color, #3498db)"
                : "none",
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
            zIndex: 100,
        }),
        singleValue: (provided) => ({
            ...provided,
            color: "var(--primary-text-color, #333)",
        }),
    };

    // Financial type options
    const financialTypeOptions = [
        { value: "", label: "Todos os Tipos" },
        { value: "unico", label: "Financeiro Único" },
        { value: "recorrente", label: "Recorrente" },
        { value: "personalizado", label: "Personalizado" },
    ];

    // Render period card

    const renderFinancialBreakdownContent = (month) => {
        const now = new Date();

        // Handle both "Mês Ano" and "Ano" formats
        let monthName, year, monthNum, isPast;

        if (month.period_label.includes(" ")) {
            // Monthly format: "Janeiro 2024"
            [monthName, year] = month.period_label.split(" ");
            const monthNames = [
                "Janeiro",
                "Fevereiro",
                "Março",
                "Abril",
                "Maio",
                "Junho",
                "Julho",
                "Agosto",
                "Setembro",
                "Outubro",
                "Novembro",
                "Dezembro",
            ];
            monthNum = monthNames.indexOf(monthName) + 1;
            isPast =
                Number(year) < now.getFullYear() ||
                (Number(year) == now.getFullYear() &&
                    monthNum < now.getMonth() + 1);
        } else {
            // Yearly format: "2024"
            year = month.period_label;
            monthName = null;
            monthNum = null;
            isPast = Number(year) < now.getFullYear();
        }

        const isConcluded =
            month.pending_count === 0 && month.overdue_count === 0;

        // "A receber" deve ser diminuído pelo recebido e atrasado (aka pending_amount)
        const aReceber = month.pending_amount;

        return (
            <>
                {isPast
                    ? month.overdue_amount > 0 && (
                        <div className="financial-breakdown-row financial-breakdown-overdue">
                            <span>A Receber (Atrasado)</span>
                            <span className="financial-value-overdue">
                                {financialApi.formatCurrency(
                                    month.overdue_amount,
                                )}
                            </span>
                        </div>
                    )
                    : aReceber > 0 && (
                        <div className="financial-breakdown-row">
                            <span>A Receber</span>
                            <span className="financial-value-pending">
                                {financialApi.formatCurrency(aReceber)}
                            </span>
                        </div>
                    )}
                {month.already_received > 0 && (
                    <div className="financial-breakdown-row">
                        <span>Recebido</span>
                        <span className="financial-value-received">
                            {financialApi.formatCurrency(
                                month.already_received,
                            )}
                        </span>
                    </div>
                )}
                {!isPast && month.overdue_amount > 0 && (
                    <div className="financial-breakdown-row financial-breakdown-overdue">
                        <span>Atrasado</span>
                        <span className="financial-value-overdue">
                            {financialApi.formatCurrency(month.overdue_amount)}
                        </span>
                    </div>
                )}
                <div className="financial-breakdown-row financial-breakdown-total">
                    <span>
                        {isPast && isConcluded
                            ? "Total Recebido"
                            : "Total a receber"}
                    </span>
                    <span>
                        {financialApi.formatCurrency(month.total_to_receive)}
                    </span>
                </div>
            </>
        );
    };

    // Generate custom period breakdown
    const getCustomPeriodBreakdown = () => {
        if (
            !detailedSummary?.monthly_breakdown ||
            !customStartDate ||
            !customEndDate
        ) {
            return [];
        }

        // Parse month inputs (format: YYYY-MM)
        const [startYear, startMonth] = customStartDate.split("-").map(Number);
        const [endYear, endMonth] = customEndDate.split("-").map(Number);

        // Create dates - first day of start month, last day of end month
        const startDate = new Date(startYear, startMonth - 1, 1);
        const endDate = new Date(endYear, endMonth, 0); // Day 0 = last day of previous month
        const results = [];

        if (customPeriodType === "mensal") {
            // Generate monthly cards for the period
            let currentDate = new Date(startDate);
            while (currentDate <= endDate) {
                const year = currentDate.getFullYear().toString();
                const monthNames = [
                    "Janeiro",
                    "Fevereiro",
                    "Março",
                    "Abril",
                    "Maio",
                    "Junho",
                    "Julho",
                    "Agosto",
                    "Setembro",
                    "Outubro",
                    "Novembro",
                    "Dezembro",
                ];
                const monthName = monthNames[currentDate.getMonth()];
                const periodLabel = `${monthName} ${year}`;

                // Find matching month in breakdown
                const monthData = detailedSummary.monthly_breakdown.find(
                    (m) => m.period_label === periodLabel,
                );

                if (monthData) {
                    results.push(monthData);
                }

                currentDate.setMonth(currentDate.getMonth() + 1);
            }
        } else if (customPeriodType === "anual") {
            // Generate yearly aggregated cards
            const yearlyData = {};

            for (let year = startYear; year <= endYear; year++) {
                const yearStr = year.toString();
                yearlyData[yearStr] = {
                    year: yearStr,
                    total_to_receive: 0,
                    already_received: 0,
                    pending_amount: 0,
                    overdue_amount: 0,
                    paid_count: 0,
                    pending_count: 0,
                    overdue_count: 0,
                    period_label: yearStr,
                    period: yearStr, // Add period property for key
                };

                detailedSummary.monthly_breakdown
                    .filter((m) => m.period_label.endsWith(yearStr))
                    .forEach((month) => {
                        yearlyData[yearStr].total_to_receive +=
                            month.total_to_receive || 0;
                        yearlyData[yearStr].already_received +=
                            month.already_received || 0;
                        yearlyData[yearStr].pending_amount +=
                            month.pending_amount || 0;
                        yearlyData[yearStr].overdue_amount +=
                            month.overdue_amount || 0;
                        yearlyData[yearStr].paid_count += month.paid_count || 0;
                        yearlyData[yearStr].pending_count +=
                            month.pending_count || 0;
                        yearlyData[yearStr].overdue_count +=
                            month.overdue_count || 0;
                    });
            }

            results.push(...Object.values(yearlyData));
        }

        return results;
    };

    if (loading) {
        return (
            <div className="financial-container">
                <div className="financial-loading">
                    <p className="financial-loading-text">
                        Carregando financeiro...
                    </p>
                </div>
            </div>
        );
    }

    return (
        <div className="financial-container">
            {/* Header */}
            <div className="financial-header">
                <h1 className="financial-title">🪙 Financeiro</h1>
            </div>

            {/* Error */}
            {error && <div className="financial-error">{error}</div>}

            {/* Main Page Tabs */}
            <div className="financial-main-tabs">
                <button
                    className={`financial-main-tab ${mainTab === "dashboard" ? "active" : ""}`}
                    onClick={() => setMainTab("dashboard")}
                >
                    📊 Painel Principal
                </button>
                <button
                    className={`financial-main-tab ${mainTab === "table" ? "active" : ""}`}
                    onClick={() => setMainTab("table")}
                >
                    📋 Tabela de Financeiros
                </button>
            </div>

            {/* Dashboard Tab */}
            {mainTab === "dashboard" && (
                <>
                    {/* Visão por Período - with tabs */}
                    {detailedLoading && !detailedSummary && (
                        <div className="financial-periods-section financial-gradient-section">
                            <h2 className="financial-section-title">
                                📅 Visão por Período
                            </h2>
                            <p
                                style={{
                                    textAlign: "center",
                                    padding: "2rem",
                                    color: "var(--text-secondary, #888)",
                                }}
                            >
                                Carregando detalhamento financeiro...
                            </p>
                        </div>
                    )}
                    {detailedSummary && (
                        <div className="financial-periods-section financial-gradient-section">
                            <h2 className="financial-section-title">
                                📅 Visão por Período
                            </h2>
                            <div className="financial-tabs">
                                <button
                                    className={`financial-tab ${periodTab === "monthlyExpanded" ? "active" : ""}`}
                                    onClick={() =>
                                        setPeriodTab("monthlyExpanded")
                                    }
                                >
                                    Mês Atual
                                </button>
                                <button
                                    className={`financial-tab ${periodTab === "currentYear" ? "active" : ""}`}
                                    onClick={() => setPeriodTab("currentYear")}
                                >
                                    Ano Atual
                                </button>
                                <button
                                    className={`financial-tab ${periodTab === "yearly" ? "active" : ""}`}
                                    onClick={() => setPeriodTab("yearly")}
                                >
                                    Anual
                                </button>
                                <button
                                    className={`financial-tab ${periodTab === "customRange" ? "active" : ""}`}
                                    onClick={() => setPeriodTab("customRange")}
                                >
                                    📌 Personalizado
                                </button>
                                <button
                                    className={`financial-tab ${periodTab === "allTime" ? "active" : ""}`}
                                    onClick={() => setPeriodTab("allTime")}
                                >
                                    Total Completo
                                </button>
                            </div>

                            <div className="financial-tabs-content">
                                {periodTab === "allTime" && (
                                    <div className="financial-main-summary">
                                        <div className="financial-summary-card financial-summary-total-pending">
                                            <div className="financial-summary-icon">
                                                ⏳
                                            </div>
                                            <div className="financial-summary-content">
                                                <div className="financial-summary-value">
                                                    {financialApi.formatCurrency(
                                                        detailedSummary.total_pending,
                                                    )}
                                                </div>
                                                <div className="financial-summary-label">
                                                    Total Pendente
                                                </div>
                                                <div className="financial-summary-count">
                                                    a receber
                                                </div>
                                            </div>
                                        </div>

                                        <div className="financial-summary-card financial-summary-received">
                                            <div className="financial-summary-icon">
                                                ✅
                                            </div>
                                            <div className="financial-summary-content">
                                                <div className="financial-summary-value">
                                                    {financialApi.formatCurrency(
                                                        detailedSummary.total_received,
                                                    )}
                                                </div>
                                                <div className="financial-summary-label">
                                                    Total Recebido
                                                </div>
                                                <div className="financial-summary-count">
                                                    já pago
                                                </div>
                                            </div>
                                        </div>

                                        <div className="financial-summary-card financial-summary-overdue">
                                            <div className="financial-summary-icon">
                                                ⚠️
                                            </div>
                                            <div className="financial-summary-content">
                                                <div className="financial-summary-value">
                                                    {financialApi.formatCurrency(
                                                        detailedSummary.total_overdue,
                                                    )}
                                                </div>
                                                <div className="financial-summary-label">
                                                    Em Atraso
                                                </div>
                                                <div className="financial-summary-count">
                                                    {
                                                        detailedSummary.total_overdue_count
                                                    }{" "}
                                                    parcelas
                                                </div>
                                            </div>
                                        </div>

                                        <div className="financial-summary-card financial-summary-total">
                                            <div className="financial-summary-icon">
                                                💵
                                            </div>
                                            <div className="financial-summary-content">
                                                <div className="financial-summary-value">
                                                    {financialApi.formatCurrency(
                                                        detailedSummary.total_to_receive,
                                                    )}
                                                </div>
                                                <div className="financial-summary-label">
                                                    Total Geral
                                                </div>
                                                <div className="financial-summary-count">
                                                    todos os contratos
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                )}

                                {periodTab === "monthlyExpanded" &&
                                    detailedSummary.monthly_breakdown && (
                                        <div
                                            key={periodTab}
                                            className="financial-breakdown-grid financial-breakdown-grid-monthly"
                                        >
                                            {detailedSummary.monthly_breakdown
                                                .slice(-7)
                                                .slice(1, -1)
                                                .map((month) => {
                                                    const now = new Date();
                                                    const [monthName, year] =
                                                        month.period_label.split(
                                                            " ",
                                                        );
                                                    const monthNames = [
                                                        "Janeiro",
                                                        "Fevereiro",
                                                        "Março",
                                                        "Abril",
                                                        "Maio",
                                                        "Junho",
                                                        "Julho",
                                                        "Agosto",
                                                        "Setembro",
                                                        "Outubro",
                                                        "Novembro",
                                                        "Dezembro",
                                                    ];
                                                    const monthNum =
                                                        monthNames.indexOf(
                                                            monthName,
                                                        ) + 1;
                                                    const isCurrent =
                                                        year ==
                                                        now.getFullYear() &&
                                                        monthNum - 1 ==
                                                        now.getMonth();
                                                    const isPast =
                                                        year <
                                                        now.getFullYear() ||
                                                        (year ==
                                                            now.getFullYear() &&
                                                            monthNum <
                                                            now.getMonth() +
                                                            1);
                                                    return (
                                                        <div
                                                            key={month.period}
                                                            className={`financial-breakdown-item ${isCurrent ? "current-month" : ""}`}
                                                        >
                                                            <div className="financial-breakdown-month">
                                                                {
                                                                    month.period_label
                                                                }
                                                            </div>
                                                            <div className="financial-breakdown-values">
                                                                {renderFinancialBreakdownContent(
                                                                    month,
                                                                )}
                                                            </div>
                                                            <div className="financial-breakdown-counts">
                                                                {
                                                                    month.paid_count
                                                                }{" "}
                                                                pagos ·{" "}
                                                                {isPast
                                                                    ? month.overdue_count
                                                                    : month.pending_count}{" "}
                                                                {isPast
                                                                    ? "atrasados"
                                                                    : "pendentes"}
                                                                {!isPast &&
                                                                    month.overdue_count >
                                                                    0 &&
                                                                    ` · ${month.overdue_count} atrasados`}
                                                            </div>
                                                        </div>
                                                    );
                                                })}
                                        </div>
                                    )}

                                {periodTab === "currentYear" &&
                                    detailedSummary.monthly_breakdown && (
                                        <div
                                            key={periodTab}
                                            className="financial-breakdown-grid financial-breakdown-grid-current-year"
                                        >
                                            {(() => {
                                                const currentYear = new Date()
                                                    .getFullYear()
                                                    .toString();
                                                const monthNames = [
                                                    "Janeiro",
                                                    "Fevereiro",
                                                    "Março",
                                                    "Abril",
                                                    "Maio",
                                                    "Junho",
                                                    "Julho",
                                                    "Agosto",
                                                    "Setembro",
                                                    "Outubro",
                                                    "Novembro",
                                                    "Dezembro",
                                                ];
                                                const monthsInYear = {};
                                                monthNames.forEach(
                                                    (name, index) => {
                                                        monthsInYear[name] =
                                                            null;
                                                    },
                                                );
                                                detailedSummary.monthly_breakdown
                                                    .filter(
                                                        (month) =>
                                                            month.period_label.split(
                                                                " ",
                                                            )[1] ===
                                                            currentYear,
                                                    )
                                                    .forEach((month) => {
                                                        const monthName =
                                                            month.period_label.split(
                                                                " ",
                                                            )[0];
                                                        monthsInYear[
                                                            monthName
                                                        ] = month;
                                                    });
                                                return monthNames.map(
                                                    (name) => {
                                                        const month =
                                                            monthsInYear[name];
                                                        const now = new Date();
                                                        const isCurrent =
                                                            name ===
                                                            monthNames[
                                                            new Date().getMonth()
                                                            ];
                                                        const isPast =
                                                            currentYear <
                                                            now.getFullYear() ||
                                                            (currentYear ==
                                                                now.getFullYear() &&
                                                                monthNames.indexOf(
                                                                    name,
                                                                ) +
                                                                1 <
                                                                now.getMonth() +
                                                                1);
                                                        if (!month) {
                                                            return (
                                                                <div
                                                                    key={name}
                                                                    className={`financial-breakdown-item ${isCurrent ? "current-month" : ""}`}
                                                                >
                                                                    <div className="financial-breakdown-month">
                                                                        {name}{" "}
                                                                        {
                                                                            currentYear
                                                                        }
                                                                    </div>
                                                                    <div className="financial-breakdown-values">
                                                                        <div className="financial-breakdown-row">
                                                                            <span>
                                                                                A
                                                                                Receber:
                                                                            </span>
                                                                            <span className="financial-value-pending">
                                                                                R$
                                                                                0,00
                                                                            </span>
                                                                        </div>
                                                                        <div className="financial-breakdown-row">
                                                                            <span>
                                                                                Recebido:
                                                                            </span>
                                                                            <span className="financial-value-received">
                                                                                R$
                                                                                0,00
                                                                            </span>
                                                                        </div>
                                                                    </div>
                                                                    <div className="financial-breakdown-counts">
                                                                        0 pagos
                                                                        · 0
                                                                        {isPast
                                                                            ? " atrasados"
                                                                            : " pendentes"}
                                                                    </div>
                                                                </div>
                                                            );
                                                        }
                                                        return (
                                                            <div
                                                                key={
                                                                    month.period
                                                                }
                                                                className={`financial-breakdown-item ${isCurrent ? "current-month" : ""}`}
                                                            >
                                                                <div className="financial-breakdown-month">
                                                                    {
                                                                        month.period_label
                                                                    }
                                                                </div>
                                                                <div className="financial-breakdown-values">
                                                                    {renderFinancialBreakdownContent(
                                                                        month,
                                                                    )}
                                                                </div>
                                                                <div className="financial-breakdown-counts">
                                                                    {
                                                                        month.paid_count
                                                                    }{" "}
                                                                    pagos ·{" "}
                                                                    {isPast
                                                                        ? month.overdue_count
                                                                        : month.pending_count}{" "}
                                                                    {isPast
                                                                        ? "atrasados"
                                                                        : "pendentes"}
                                                                    {!isPast &&
                                                                        month.overdue_count >
                                                                        0 &&
                                                                        ` · ${month.overdue_count} atrasados`}
                                                                </div>
                                                            </div>
                                                        );
                                                    },
                                                );
                                            })()}
                                        </div>
                                    )}

                                {periodTab === "customRange" && (
                                    <div className="financial-breakdown-grid">
                                        <div
                                            style={{
                                                gridColumn: "1 / -1",
                                                display: "flex",
                                                gap: "16px",
                                                marginBottom: "20px",
                                                alignItems: "flex-end",
                                                flexWrap: "wrap",
                                            }}
                                        >
                                            <div
                                                style={{
                                                    display: "flex",
                                                    gap: "8px",
                                                    alignItems: "center",
                                                }}
                                            >
                                                <label
                                                    style={{
                                                        fontSize: "14px",
                                                        fontWeight: "500",
                                                    }}
                                                >
                                                    Tipo:
                                                </label>
                                                <select
                                                    value={customPeriodType}
                                                    onChange={(e) =>
                                                        setCustomPeriodType(
                                                            e.target.value,
                                                        )
                                                    }
                                                    style={{
                                                        padding: "8px 12px",
                                                        borderRadius: "6px",
                                                        border: "1px solid var(--border-color, #ddd)",
                                                        backgroundColor:
                                                            "var(--content-bg, white)",
                                                        color: "var(--primary-text-color, #2c3e50)",
                                                        cursor: "pointer",
                                                        fontSize: "14px",
                                                    }}
                                                >
                                                    <option value="mensal">
                                                        Mensal
                                                    </option>
                                                    <option value="anual">
                                                        Anual
                                                    </option>
                                                </select>
                                            </div>

                                            <div
                                                style={{
                                                    display: "flex",
                                                    gap: "8px",
                                                    alignItems: "center",
                                                }}
                                            >
                                                <label
                                                    style={{
                                                        fontSize: "14px",
                                                        fontWeight: "500",
                                                    }}
                                                >
                                                    Mês Inicial:
                                                </label>
                                                <input
                                                    type="month"
                                                    value={customStartDate}
                                                    onChange={(e) =>
                                                        setCustomStartDate(
                                                            e.target.value,
                                                        )
                                                    }
                                                    style={{
                                                        padding: "8px 12px",
                                                        borderRadius: "6px",
                                                        border: "1px solid var(--border-color, #ddd)",
                                                        backgroundColor:
                                                            "var(--content-bg, white)",
                                                        color: "var(--primary-text-color, #2c3e50)",
                                                        cursor: "pointer",
                                                        fontSize: "14px",
                                                    }}
                                                />
                                            </div>

                                            <div
                                                style={{
                                                    display: "flex",
                                                    gap: "8px",
                                                    alignItems: "center",
                                                }}
                                            >
                                                <label
                                                    style={{
                                                        fontSize: "14px",
                                                        fontWeight: "500",
                                                    }}
                                                >
                                                    Mês Final:
                                                </label>
                                                <input
                                                    type="month"
                                                    value={customEndDate}
                                                    onChange={(e) =>
                                                        setCustomEndDate(
                                                            e.target.value,
                                                        )
                                                    }
                                                    style={{
                                                        padding: "8px 12px",
                                                        borderRadius: "6px",
                                                        border: "1px solid var(--border-color, #ddd)",
                                                        backgroundColor:
                                                            "var(--content-bg, white)",
                                                        color: "var(--primary-text-color, #2c3e50)",
                                                        cursor: "pointer",
                                                        fontSize: "14px",
                                                    }}
                                                />
                                            </div>
                                        </div>

                                        {customStartDate && customEndDate ? (
                                            (() => {
                                                const breakdown =
                                                    getCustomPeriodBreakdown();
                                                return breakdown.length > 0 ? (
                                                    <div
                                                        key={periodTab}
                                                        className="financial-breakdown-grid financial-breakdown-grid-flexible"
                                                    >
                                                        {breakdown.map(
                                                            (month) => {
                                                                const now =
                                                                    new Date();

                                                                // Handle both "Mês Ano" and "Ano" formats
                                                                let monthName,
                                                                    year,
                                                                    monthNum,
                                                                    isCurrent,
                                                                    isPast;

                                                                if (
                                                                    month.period_label.includes(
                                                                        " ",
                                                                    )
                                                                ) {
                                                                    // Monthly format: "Janeiro 2024"
                                                                    [
                                                                        monthName,
                                                                        year,
                                                                    ] =
                                                                        month.period_label.split(
                                                                            " ",
                                                                        );
                                                                    const monthNames =
                                                                        [
                                                                            "Janeiro",
                                                                            "Fevereiro",
                                                                            "Março",
                                                                            "Abril",
                                                                            "Maio",
                                                                            "Junho",
                                                                            "Julho",
                                                                            "Agosto",
                                                                            "Setembro",
                                                                            "Outubro",
                                                                            "Novembro",
                                                                            "Dezembro",
                                                                        ];
                                                                    monthNum =
                                                                        monthNames.indexOf(
                                                                            monthName,
                                                                        ) + 1;
                                                                    isCurrent =
                                                                        Number(
                                                                            year,
                                                                        ) ==
                                                                        now.getFullYear() &&
                                                                        monthNum -
                                                                        1 ==
                                                                        now.getMonth();
                                                                    isPast =
                                                                        Number(
                                                                            year,
                                                                        ) <
                                                                        now.getFullYear() ||
                                                                        (Number(
                                                                            year,
                                                                        ) ==
                                                                            now.getFullYear() &&
                                                                            monthNum <
                                                                            now.getMonth() +
                                                                            1);
                                                                } else {
                                                                    // Yearly format: "2024"
                                                                    year =
                                                                        month.period_label;
                                                                    monthName =
                                                                        null;
                                                                    monthNum =
                                                                        null;
                                                                    isCurrent =
                                                                        Number(
                                                                            year,
                                                                        ) ==
                                                                        now.getFullYear();
                                                                    isPast =
                                                                        Number(
                                                                            year,
                                                                        ) <
                                                                        now.getFullYear();
                                                                }
                                                                return (
                                                                    <div
                                                                        key={
                                                                            month.period ||
                                                                            month.period_label
                                                                        }
                                                                        className={`financial-breakdown-item ${isCurrent ? "current-month" : ""}`}
                                                                    >
                                                                        <div className="financial-breakdown-month">
                                                                            {
                                                                                month.period_label
                                                                            }
                                                                        </div>
                                                                        <div className="financial-breakdown-values">
                                                                            {renderFinancialBreakdownContent(
                                                                                month,
                                                                            )}
                                                                        </div>
                                                                        <div className="financial-breakdown-counts">
                                                                            {
                                                                                month.paid_count
                                                                            }{" "}
                                                                            pagos
                                                                            ·{" "}
                                                                            {isPast
                                                                                ? month.overdue_count
                                                                                : month.pending_count}{" "}
                                                                            {isPast
                                                                                ? "atrasados"
                                                                                : "pendentes"}
                                                                            {!isPast &&
                                                                                month.overdue_count >
                                                                                0 &&
                                                                                ` · ${month.overdue_count} atrasados`}
                                                                        </div>
                                                                    </div>
                                                                );
                                                            },
                                                        )}
                                                    </div>
                                                ) : (
                                                    <div
                                                        style={{
                                                            gridColumn:
                                                                "1 / -1",
                                                            padding: "20px",
                                                            textAlign: "center",
                                                            color: "var(--secondary-text-color, #7f8c8d)",
                                                        }}
                                                    >
                                                        Nenhum dado disponível
                                                        para o período
                                                        selecionado.
                                                    </div>
                                                );
                                            })()
                                        ) : (
                                            <div
                                                style={{
                                                    gridColumn: "1 / -1",
                                                    padding: "20px",
                                                    textAlign: "center",
                                                    color: "var(--secondary-text-color, #7f8c8d)",
                                                }}
                                            >
                                                Selecione os meses inicial e
                                                final para visualizar os dados.
                                            </div>
                                        )}
                                    </div>
                                )}

                                {periodTab === "yearly" &&
                                    detailedSummary.monthly_breakdown && (
                                        <div
                                            key={periodTab}
                                            className="financial-breakdown-grid financial-breakdown-grid-flexible"
                                        >
                                            {/* Group by year - simple implementation */}
                                            {(() => {
                                                const yearlyData = {};
                                                detailedSummary.monthly_breakdown.forEach(
                                                    (month) => {
                                                        const year =
                                                            month.period_label.split(
                                                                " ",
                                                            )[1];
                                                        if (!yearlyData[year]) {
                                                            yearlyData[year] = {
                                                                year,
                                                                total_to_receive: 0,
                                                                already_received: 0,
                                                                pending_amount: 0,
                                                                overdue_amount: 0,
                                                                paid_count: 0,
                                                                pending_count: 0,
                                                                overdue_count: 0,
                                                            };
                                                        }
                                                        yearlyData[
                                                            year
                                                        ].total_to_receive +=
                                                            month.total_to_receive ||
                                                            0;
                                                        yearlyData[
                                                            year
                                                        ].already_received +=
                                                            month.already_received ||
                                                            0;
                                                        yearlyData[
                                                            year
                                                        ].pending_amount +=
                                                            month.pending_amount ||
                                                            0;
                                                        yearlyData[
                                                            year
                                                        ].overdue_amount +=
                                                            month.overdue_amount ||
                                                            0;
                                                        yearlyData[
                                                            year
                                                        ].paid_count +=
                                                            month.paid_count ||
                                                            0;
                                                        yearlyData[
                                                            year
                                                        ].pending_count +=
                                                            month.pending_count ||
                                                            0;
                                                        yearlyData[
                                                            year
                                                        ].overdue_count +=
                                                            month.overdue_count ||
                                                            0;
                                                    },
                                                );
                                                return Object.values(
                                                    yearlyData,
                                                ).map((yearData) => {
                                                    const now = new Date();
                                                    const isCurrent =
                                                        yearData.year ===
                                                        now
                                                            .getFullYear()
                                                            .toString();
                                                    const isPast =
                                                        yearData.year <
                                                        now.getFullYear();
                                                    return (
                                                        <div
                                                            key={yearData.year}
                                                            className={`financial-breakdown-item ${isCurrent ? "current-month" : ""}`}
                                                        >
                                                            <div className="financial-breakdown-month">
                                                                {yearData.year}
                                                            </div>
                                                            <div className="financial-breakdown-values">
                                                                {(() => {
                                                                    const now =
                                                                        new Date();
                                                                    const isPast =
                                                                        yearData.year <
                                                                        now.getFullYear();

                                                                    const aReceber =
                                                                        yearData.pending_amount;
                                                                    const isConcluded =
                                                                        yearData.pending_count ===
                                                                        0 &&
                                                                        yearData.overdue_count ===
                                                                        0;

                                                                    return (
                                                                        <>
                                                                            {isPast
                                                                                ? yearData.overdue_amount >
                                                                                0 && (
                                                                                    <div className="financial-breakdown-row financial-breakdown-overdue">
                                                                                        <span>
                                                                                            A
                                                                                            Receber
                                                                                            (Atrasado)
                                                                                        </span>
                                                                                        <span className="financial-value-overdue">
                                                                                            {financialApi.formatCurrency(
                                                                                                yearData.overdue_amount,
                                                                                            )}
                                                                                        </span>
                                                                                    </div>
                                                                                )
                                                                                : aReceber >
                                                                                0 && (
                                                                                    <div className="financial-breakdown-row">
                                                                                        <span>
                                                                                            A
                                                                                            Receber
                                                                                        </span>
                                                                                        <span className="financial-value-pending">
                                                                                            {financialApi.formatCurrency(
                                                                                                aReceber,
                                                                                            )}
                                                                                        </span>
                                                                                    </div>
                                                                                )}
                                                                            {yearData.already_received >
                                                                                0 && (
                                                                                    <div className="financial-breakdown-row">
                                                                                        <span>
                                                                                            Recebido
                                                                                        </span>
                                                                                        <span className="financial-value-received">
                                                                                            {financialApi.formatCurrency(
                                                                                                yearData.already_received,
                                                                                            )}
                                                                                        </span>
                                                                                    </div>
                                                                                )}
                                                                            {!isPast &&
                                                                                yearData.overdue_amount >
                                                                                0 && (
                                                                                    <div className="financial-breakdown-row financial-breakdown-overdue">
                                                                                        <span>
                                                                                            Atrasado
                                                                                        </span>
                                                                                        <span className="financial-value-overdue">
                                                                                            {financialApi.formatCurrency(
                                                                                                yearData.overdue_amount,
                                                                                            )}
                                                                                        </span>
                                                                                    </div>
                                                                                )}
                                                                            <div className="financial-breakdown-row financial-breakdown-total">
                                                                                <span>
                                                                                    {isPast &&
                                                                                        isConcluded
                                                                                        ? "Total Recebido"
                                                                                        : "Total a receber"}
                                                                                </span>
                                                                                <span>
                                                                                    {financialApi.formatCurrency(
                                                                                        yearData.total_to_receive,
                                                                                    )}
                                                                                </span>
                                                                            </div>
                                                                        </>
                                                                    );
                                                                })()}
                                                            </div>
                                                            <div className="financial-breakdown-counts">
                                                                {
                                                                    yearData.paid_count
                                                                }{" "}
                                                                pagos ·{" "}
                                                                {isPast
                                                                    ? yearData.overdue_count
                                                                    : yearData.pending_count}{" "}
                                                                {isPast
                                                                    ? "atrasados"
                                                                    : "pendentes"}
                                                                {!isPast &&
                                                                    yearData.overdue_count >
                                                                    0 &&
                                                                    ` · ${yearData.overdue_count} atrasados`}
                                                            </div>
                                                        </div>
                                                    );
                                                });
                                            })()}
                                        </div>
                                    )}
                            </div>
                        </div>
                    )}

                    {/* Tabs for Overdue/Upcoming - Overdue first */}
                    <div className="financial-tabs-section financial-gradient-section">
                        <h2 className="financial-section-title">
                            💰 Vencimentos
                        </h2>
                        <div className="financial-tabs">
                            <button
                                className={`financial-tab ${activeTab === "overdue" ? "active" : ""}`}
                                onClick={() => setActiveTab("overdue")}
                            >
                                ⚠️ Em Atraso ({overdueFinancials.length})
                            </button>
                            <button
                                className={`financial-tab ${activeTab === "upcoming" ? "active" : ""}`}
                                onClick={() => setActiveTab("upcoming")}
                            >
                                📅 Próximos Vencimentos (
                                {upcomingFinancials.length})
                            </button>
                        </div>

                        <div className="financial-tabs-content">
                            {activeTab === "overdue" && (
                                <div className="financial-overdue-list">
                                    {overdueFinancials.length === 0 ? (
                                        <p className="financial-empty-message">
                                            Nenhum financeiro em atraso. 🎉
                                        </p>
                                    ) : (
                                        <>
                                            <table className="financial-mini-table financial-mini-table-overdue">
                                                <thead>
                                                    <tr>
                                                        <th>
                                                            {config.labels
                                                                ?.client ||
                                                                "Cliente"}
                                                        </th>
                                                        <th>
                                                            {config.labels
                                                                ?.contract ||
                                                                "Contrato"}
                                                        </th>
                                                        <th>Parcela</th>
                                                        <th>Valor</th>
                                                        <th>Vencimento</th>
                                                        <th>Atraso</th>
                                                        <th>Ações</th>
                                                    </tr>
                                                </thead>
                                                <tbody>
                                                    {overdueFinancials
                                                        .slice(
                                                            (overdueCurrentPage -
                                                                1) *
                                                            overdueItemsPerPage,
                                                            overdueCurrentPage *
                                                            overdueItemsPerPage,
                                                        )
                                                        .map((financial) => {
                                                            const dueDate =
                                                                financial.due_date
                                                                    ? new Date(
                                                                        financial.due_date,
                                                                    )
                                                                    : null;
                                                            const today =
                                                                new Date();
                                                            today.setHours(
                                                                0,
                                                                0,
                                                                0,
                                                                0,
                                                            );
                                                            const diffDays =
                                                                dueDate
                                                                    ? Math.abs(
                                                                        Math.ceil(
                                                                            (today -
                                                                                dueDate) /
                                                                            (1000 *
                                                                                60 *
                                                                                60 *
                                                                                24),
                                                                        ),
                                                                    )
                                                                    : null;

                                                            return (
                                                                <tr
                                                                    key={
                                                                        financial.installment_id
                                                                    }
                                                                >
                                                                    <td>
                                                                        {
                                                                            financial.client_name
                                                                        }
                                                                    </td>
                                                                    <td>
                                                                        {financial.contract_model ||
                                                                            "—"}
                                                                    </td>
                                                                    <td>
                                                                        {financial.installment_label ||
                                                                            "—"}
                                                                    </td>
                                                                    <td>
                                                                        {financialApi.formatCurrency(
                                                                            financial.received_value,
                                                                        )}
                                                                    </td>
                                                                    <td className="financial-overdue-date">
                                                                        {dueDate
                                                                            ? dueDate.toLocaleDateString(
                                                                                "pt-BR",
                                                                            )
                                                                            : "—"}
                                                                    </td>
                                                                    <td>
                                                                        <span className="financial-days-badge financial-days-overdue">
                                                                            {diffDays !==
                                                                                null
                                                                                ? `${diffDays} dias`
                                                                                : "—"}
                                                                        </span>
                                                                    </td>
                                                                    <td>
                                                                        <div className="financial-action-group">
                                                                            <button
                                                                                className="financial-action-btn financial-action-pay"
                                                                                onClick={(
                                                                                    e,
                                                                                ) =>
                                                                                    handleMarkPaid(
                                                                                        financial.installment_id,
                                                                                        financial.contract_financial_id,
                                                                                        e,
                                                                                    )
                                                                                }
                                                                                title="Marcar como pago"
                                                                            >
                                                                                🤑
                                                                            </button>
                                                                            <button
                                                                                className="financial-action-btn financial-action-view"
                                                                                onClick={(
                                                                                    e,
                                                                                ) => {
                                                                                    console.log(
                                                                                        "Financial object:",
                                                                                        financial,
                                                                                    );
                                                                                    console.log(
                                                                                        "Available financial records:",
                                                                                        financialRecords.map(
                                                                                            (
                                                                                                f,
                                                                                            ) => ({
                                                                                                id: f.id,
                                                                                                contract_id:
                                                                                                    f.contract_id,
                                                                                            }),
                                                                                        ),
                                                                                    );
                                                                                    console.log(
                                                                                        "Looking for contract_financial_id:",
                                                                                        financial.contract_financial_id,
                                                                                    );

                                                                                    const financialRecord =
                                                                                        financialRecords.find(
                                                                                            (
                                                                                                f,
                                                                                            ) =>
                                                                                                f.id ===
                                                                                                financial.contract_financial_id,
                                                                                        );

                                                                                    console.log(
                                                                                        "Found financial record:",
                                                                                        financialRecord,
                                                                                    );

                                                                                    if (
                                                                                        financialRecord
                                                                                    ) {
                                                                                        openFinancialModal(
                                                                                            financialRecord,
                                                                                            e,
                                                                                        );
                                                                                    } else {
                                                                                        alert(
                                                                                            "Financial record not found. Check console for details.",
                                                                                        );
                                                                                    }
                                                                                }}
                                                                                title="Ver detalhes"
                                                                            >
                                                                                👁️
                                                                            </button>
                                                                        </div>
                                                                    </td>
                                                                </tr>
                                                            );
                                                        })}
                                                </tbody>
                                            </table>
                                            {overdueFinancials.length > 0 && (
                                                <div
                                                    className="financial-pagination"
                                                    style={{
                                                        borderTop:
                                                            "1px solid var(--border-color, #eee)",
                                                        marginTop: "16px",
                                                    }}
                                                >
                                                    <Pagination
                                                        currentPage={
                                                            overdueCurrentPage
                                                        }
                                                        totalItems={
                                                            overdueFinancials.length
                                                        }
                                                        itemsPerPage={
                                                            overdueItemsPerPage
                                                        }
                                                        onPageChange={
                                                            setOverdueCurrentPage
                                                        }
                                                        onItemsPerPageChange={
                                                            setOverdueItemsPerPage
                                                        }
                                                    />
                                                </div>
                                            )}
                                        </>
                                    )}
                                </div>
                            )}

                            {activeTab === "upcoming" && (
                                <div className="financial-upcoming-list">
                                    {upcomingFinancials.length === 0 ? (
                                        <p className="financial-empty-message">
                                            Nenhum financeiro próximo do
                                            vencimento nos próximos 30 dias.
                                        </p>
                                    ) : (
                                        <>
                                            <table className="financial-mini-table">
                                                <thead>
                                                    <tr>
                                                        <th>
                                                            {config.labels
                                                                ?.client ||
                                                                "Cliente"}
                                                        </th>
                                                        <th>
                                                            {config.labels
                                                                ?.contract ||
                                                                "Contrato"}
                                                        </th>
                                                        <th>Parcela</th>
                                                        <th>Valor</th>
                                                        <th>Vencimento</th>
                                                        <th>Dias</th>
                                                        <th>Ações</th>
                                                    </tr>
                                                </thead>
                                                <tbody>
                                                    {upcomingFinancials
                                                        .slice(
                                                            (upcomingCurrentPage -
                                                                1) *
                                                            upcomingItemsPerPage,
                                                            upcomingCurrentPage *
                                                            upcomingItemsPerPage,
                                                        )
                                                        .map((financial) => {
                                                            const dueDate =
                                                                financial.due_date
                                                                    ? new Date(
                                                                        financial.due_date,
                                                                    )
                                                                    : null;
                                                            const today =
                                                                new Date();
                                                            today.setHours(
                                                                0,
                                                                0,
                                                                0,
                                                                0,
                                                            );
                                                            const diffDays =
                                                                dueDate
                                                                    ? Math.ceil(
                                                                        (dueDate -
                                                                            today) /
                                                                        (1000 *
                                                                            60 *
                                                                            60 *
                                                                            24),
                                                                    )
                                                                    : null;

                                                            return (
                                                                <tr
                                                                    key={
                                                                        financial.installment_id
                                                                    }
                                                                >
                                                                    <td>
                                                                        {
                                                                            financial.client_name
                                                                        }
                                                                    </td>
                                                                    <td>
                                                                        {financial.contract_model ||
                                                                            "—"}
                                                                    </td>
                                                                    <td>
                                                                        {financial.installment_label ||
                                                                            "—"}
                                                                    </td>
                                                                    <td>
                                                                        {financialApi.formatCurrency(
                                                                            financial.received_value,
                                                                        )}
                                                                    </td>
                                                                    <td>
                                                                        {dueDate
                                                                            ? dueDate.toLocaleDateString(
                                                                                "pt-BR",
                                                                            )
                                                                            : "—"}
                                                                    </td>
                                                                    <td>
                                                                        <span
                                                                            className={`financial-days-badge ${diffDays <=
                                                                                3
                                                                                ? "financial-days-urgent"
                                                                                : diffDays <=
                                                                                    7
                                                                                    ? "financial-days-soon"
                                                                                    : ""
                                                                                }`}
                                                                        >
                                                                            {diffDays !==
                                                                                null
                                                                                ? diffDays ===
                                                                                    0
                                                                                    ? "Hoje"
                                                                                    : diffDays ===
                                                                                        1
                                                                                        ? "Amanhã"
                                                                                        : `${diffDays} dias`
                                                                                : "—"}
                                                                        </span>
                                                                    </td>
                                                                    <td>
                                                                        <div className="financial-action-group">
                                                                            <button
                                                                                className="financial-action-btn financial-action-pay"
                                                                                onClick={(
                                                                                    e,
                                                                                ) =>
                                                                                    handleMarkPaid(
                                                                                        financial.installment_id,
                                                                                        financial.contract_financial_id,
                                                                                        e,
                                                                                    )
                                                                                }
                                                                                title="Marcar como pago"
                                                                            >
                                                                                🤑
                                                                            </button>
                                                                            <button
                                                                                className="financial-action-btn financial-action-view"
                                                                                onClick={(
                                                                                    e,
                                                                                ) => {
                                                                                    console.log(
                                                                                        "Financial object:",
                                                                                        financial,
                                                                                    );
                                                                                    console.log(
                                                                                        "Available financial records:",
                                                                                        financialRecords.map(
                                                                                            (
                                                                                                f,
                                                                                            ) => ({
                                                                                                id: f.id,
                                                                                                contract_id:
                                                                                                    f.contract_id,
                                                                                            }),
                                                                                        ),
                                                                                    );
                                                                                    console.log(
                                                                                        "Looking for contract_financial_id:",
                                                                                        financial.contract_financial_id,
                                                                                    );

                                                                                    const financialRecord =
                                                                                        financialRecords.find(
                                                                                            (
                                                                                                f,
                                                                                            ) =>
                                                                                                f.id ===
                                                                                                financial.contract_financial_id,
                                                                                        );

                                                                                    console.log(
                                                                                        "Found financial record:",
                                                                                        financialRecord,
                                                                                    );

                                                                                    if (
                                                                                        financialRecord
                                                                                    ) {
                                                                                        openFinancialModal(
                                                                                            financialRecord,
                                                                                            e,
                                                                                        );
                                                                                    } else {
                                                                                        alert(
                                                                                            "Financial record not found. Check console for details.",
                                                                                        );
                                                                                    }
                                                                                }}
                                                                                title="Ver detalhes"
                                                                            >
                                                                                👁️
                                                                            </button>
                                                                        </div>
                                                                    </td>
                                                                </tr>
                                                            );
                                                        })}
                                                </tbody>
                                            </table>
                                            {upcomingFinancials.length > 0 && (
                                                <div
                                                    className="financial-pagination"
                                                    style={{
                                                        borderTop:
                                                            "1px solid var(--border-color, #eee)",
                                                        marginTop: "16px",
                                                    }}
                                                >
                                                    <Pagination
                                                        currentPage={
                                                            upcomingCurrentPage
                                                        }
                                                        totalItems={
                                                            upcomingFinancials.length
                                                        }
                                                        itemsPerPage={
                                                            upcomingItemsPerPage
                                                        }
                                                        onPageChange={
                                                            setUpcomingCurrentPage
                                                        }
                                                        onItemsPerPageChange={
                                                            setUpcomingItemsPerPage
                                                        }
                                                    />
                                                </div>
                                            )}
                                        </>
                                    )}
                                </div>
                            )}
                        </div>
                    </div>
                </>
            )}

            {/* Table Tab */}
            {mainTab === "table" && (
                <>
                    {/* Filter Buttons */}
                    <div
                        className="financial-filter-buttons"
                        ref={filtersContainerRef}
                    >
                        <button
                            className={`financial-filter-button ${filter === "all" ? "active-all" : ""}`}
                            onClick={() => setFilter("all")}
                        >
                            Todos os Modelos
                        </button>
                    </div>

                    {/* Filters */}
                    <div className="financial-search-filters">
                        <Select
                            value={financialTypeOptions.find(
                                (opt) => opt.value === financialTypeFilter,
                            )}
                            onChange={(opt) =>
                                setFinancialTypeFilter(opt?.value || "")
                            }
                            options={financialTypeOptions}
                            styles={customSelectStyles}
                            placeholder="Tipo de Financeiro"
                            isClearable={false}
                        />

                        <Select
                            value={
                                clientFilter
                                    ? {
                                        value: clientFilter,
                                        label: clientFilter,
                                    }
                                    : null
                            }
                            onChange={(selected) =>
                                setClientFilter(selected ? selected.value : "")
                            }
                            options={[
                                {
                                    value: "",
                                    label: `${config.labels?.clients || "Clientes"}`,
                                },
                                ...clients
                                    .filter((c) => !c.archived_at)
                                    .map((client) => ({
                                        value: client.name,
                                        label: client.name,
                                    })),
                            ]}
                            isSearchable={true}
                            placeholder={`Filtrar por ${config.labels?.client?.toLowerCase() || "cliente"}`}
                            styles={customSelectStyles}
                        />

                        <Select
                            value={
                                contractFilter
                                    ? {
                                        value: contractFilter,
                                        label: contractFilter,
                                    }
                                    : null
                            }
                            onChange={(selected) =>
                                setContractFilter(
                                    selected ? selected.value : "",
                                )
                            }
                            options={[
                                {
                                    value: "",
                                    label: `${config.labels?.contracts || "Contratos"}`,
                                },
                                ...contracts
                                    .filter((c) => !c.archived_at)
                                    .map((contract) => ({
                                        value: contract.model,
                                        label: contract.model,
                                    })),
                            ]}
                            isSearchable={true}
                            placeholder={`Filtrar por ${config.labels?.contract?.toLowerCase() || "contrato"}`}
                            styles={customSelectStyles}
                        />

                        <Select
                            value={
                                categoryFilter
                                    ? {
                                        value: categoryFilter,
                                        label:
                                            categories.find(
                                                (c) =>
                                                    c.id === categoryFilter,
                                            )?.name || "",
                                    }
                                    : null
                            }
                            onChange={(selected) =>
                                setCategoryFilter(
                                    selected ? selected.value : "",
                                )
                            }
                            options={[
                                {
                                    value: "",
                                    label: `${config.labels?.categories || "Categorias"}`,
                                },
                                ...categories
                                    .filter((c) => !c.archived_at)
                                    .map((category) => ({
                                        value: category.id,
                                        label: category.name,
                                    })),
                            ]}
                            isSearchable={true}
                            placeholder={`Filtrar por ${config.labels?.category?.toLowerCase() || "categoria"}`}
                            styles={customSelectStyles}
                        />

                        <Select
                            value={
                                subcategoryFilter
                                    ? {
                                        value: subcategoryFilter,
                                        label:
                                            subcategories.find(
                                                (s) =>
                                                    s.id ===
                                                    subcategoryFilter,
                                            )?.name || "",
                                    }
                                    : null
                            }
                            onChange={(selected) =>
                                setSubcategoryFilter(
                                    selected ? selected.value : "",
                                )
                            }
                            options={[
                                {
                                    value: "",
                                    label: `${config.labels?.subcategories || "Subcategorias"}`,
                                },
                                ...availableSubcategories
                                    .filter((s) => !s.archived_at)
                                    .map((sub) => ({
                                        value: sub.id,
                                        label: sub.name,
                                    })),
                            ]}
                            isSearchable={true}
                            isDisabled={!categoryFilter}
                            placeholder={`Filtrar por ${config.labels?.subcategory?.toLowerCase() || "subcategoria"}`}
                            styles={customSelectStyles}
                        />

                        <div className="financial-search-row">
                            <div className="financial-date-filters">
                                <div className="financial-date-input-group">
                                    <label>De:</label>
                                    <input
                                        type="date"
                                        value={values.startDate}
                                        onChange={(e) =>
                                            setStartDate(e.target.value)
                                        }
                                        className="financial-date-input"
                                    />
                                </div>
                                <div className="financial-date-input-group">
                                    <label>Até:</label>
                                    <input
                                        type="date"
                                        value={values.endDate}
                                        onChange={(e) =>
                                            setEndDate(e.target.value)
                                        }
                                        className="financial-date-input"
                                    />
                                </div>
                            </div>

                            <input
                                type="text"
                                placeholder="Filtrar por descrição..."
                                value={descriptionFilter}
                                onChange={(e) =>
                                    setDescriptionFilter(e.target.value)
                                }
                                className="financial-search-input"
                                style={{ flex: 1 }}
                            />

                            <label className="financial-checkbox-label">
                                <input
                                    type="checkbox"
                                    checked={showCompleted}
                                    onChange={(e) =>
                                        setShowCompleted(e.target.checked)
                                    }
                                />
                                Exibir Finalizados
                            </label>

                            <button
                                className="financial-clear-filters-button"
                                onClick={clearFilters}
                            >
                                🗑️ Limpar
                            </button>
                        </div>

                        <div className="financial-search-row financial-sort-row">
                            <div className="financial-sort-group">
                                <Select
                                    value={
                                        sortBy
                                            ? {
                                                value: sortBy,
                                                label: (() => {
                                                    const sortLabels = {
                                                        client:
                                                            config.labels
                                                                ?.client ||
                                                            "Cliente",
                                                        contract:
                                                            config.labels
                                                                ?.contract ||
                                                            "Contrato",
                                                        type: "Tipo",
                                                        value: "Valor",
                                                        progress: "Progresso",
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
                                            value: "client",
                                            label:
                                                config.labels?.client ||
                                                "Cliente",
                                        },
                                        {
                                            value: "contract",
                                            label:
                                                config.labels?.contract ||
                                                "Contrato",
                                        },
                                        { value: "type", label: "Tipo" },
                                        { value: "value", label: "Valor" },
                                        {
                                            value: "progress",
                                            label: "Progresso",
                                        },
                                    ]}
                                    isSearchable={false}
                                    placeholder="Ordenar por..."
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
                        </div>
                    </div>

                    {/* Financial Table */}
                    <div className="financial-table-wrapper financial-gradient-section">
                        <div className="financial-table-header">
                            <h3 className="financial-table-header-title">
                                📋 Modelos de Financeiro (
                                {sortedFinancial.length})
                            </h3>
                            <p className="financial-table-subtitle">
                                Vencimentos
                            </p>
                        </div>

                        {paginatedFinancial.length === 0 ? (
                            <div className="financial-empty">
                                <p>Nenhum financeiro encontrado.</p>
                                <p className="financial-empty-hint">
                                    Os financeiro são configurados nos
                                    contratos. Edite um contrato para adicionar
                                    um modelo de financeiro.
                                </p>
                            </div>
                        ) : (
                            <table className="financial-table">
                                <thead>
                                    <tr>
                                        <th>
                                            {config.labels?.client || "Cliente"}
                                        </th>
                                        <th>
                                            {config.labels?.contract ||
                                                "Contrato"}
                                        </th>
                                        <th>Tipo</th>
                                        <th>
                                            Valor{" "}
                                            {config.labels?.client || "Cliente"}
                                        </th>
                                        <th>Valor Recebido</th>
                                        <th>Progresso</th>
                                        <th>Ações</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {paginatedFinancial.map((financial) => {
                                        const contractInfo =
                                            getContractInfo(financial);
                                        const progress =
                                            financial.total_installments > 0
                                                ? Math.round(
                                                    ((financial.paid_installments ||
                                                        0) /
                                                        financial.total_installments) *
                                                    100,
                                                )
                                                : 0;

                                        return (
                                            <tr key={financial.id}>
                                                <td>
                                                    <div className="financial-client-info">
                                                        <span className="financial-client-name">
                                                            {
                                                                contractInfo.clientName
                                                            }
                                                        </span>
                                                        {contractInfo.clientNickname && (
                                                            <span className="financial-client-nickname">
                                                                (
                                                                {
                                                                    contractInfo.clientNickname
                                                                }
                                                                )
                                                            </span>
                                                        )}
                                                    </div>
                                                </td>
                                                <td>{contractInfo.model}</td>
                                                <td>
                                                    <span
                                                        className={`financial-type-badge financial-type-${financial.financial_type}`}
                                                    >
                                                        {financialApi.getFinancialTypeLabel(
                                                            financial.financial_type,
                                                        )}
                                                    </span>
                                                </td>
                                                <td>
                                                    {financialApi.formatCurrency(
                                                        financial.total_client_value ||
                                                        financial.client_value,
                                                    )}
                                                </td>
                                                <td>
                                                    {financialApi.formatCurrency(
                                                        financial.total_received_value ||
                                                        financial.received_value,
                                                    )}
                                                </td>
                                                <td>
                                                    {financial.financial_type ===
                                                        "personalizado" ? (
                                                        <div className="financial-progress">
                                                            <div className="financial-progress-bar">
                                                                <div
                                                                    className="financial-progress-fill"
                                                                    style={{
                                                                        width: `${progress}%`,
                                                                    }}
                                                                />
                                                            </div>
                                                            <span className="financial-progress-text">
                                                                {financial.paid_installments ||
                                                                    0}
                                                                /
                                                                {
                                                                    financial.total_installments
                                                                }
                                                            </span>
                                                        </div>
                                                    ) : (
                                                        <span className="financial-progress-na">
                                                            —
                                                        </span>
                                                    )}
                                                </td>
                                                <td>
                                                    <button
                                                        className="financial-action-btn financial-action-view"
                                                        onClick={(e) =>
                                                            openFinancialModal(
                                                                financial,
                                                                e,
                                                            )
                                                        }
                                                        title="Ver detalhes"
                                                    >
                                                        👁️
                                                    </button>
                                                </td>
                                            </tr>
                                        );
                                    })}
                                </tbody>
                            </table>
                        )}

                        {/* Pagination */}
                        {paginatedFinancial.length > 0 && (
                            <div className="financial-pagination">
                                <Pagination
                                    currentPage={currentPage}
                                    totalItems={totalItems}
                                    itemsPerPage={itemsPerPage}
                                    onPageChange={setCurrentPage}
                                    onItemsPerPageChange={setItemsPerPage}
                                />
                            </div>
                        )}
                    </div>
                </>
            )}

            {/* Financial Details Modal */}
            {showModal &&
                selectedFinancial &&
                (() => {
                    const paidClientTotal = selectedInstallments.reduce(
                        (sum, inst) =>
                            inst.status === "pago"
                                ? sum + (inst.client_value || 0)
                                : sum,
                        0,
                    );
                    const paidReceivedTotal = selectedInstallments.reduce(
                        (sum, inst) =>
                            inst.status === "pago"
                                ? sum + (inst.received_value || 0)
                                : sum,
                        0,
                    );

                    return (
                        <div
                            className="financial-modal-overlay"
                            onClick={(e) => closeModal(e)}
                        >
                            <div
                                className="financial-modal"
                                onClick={(e) => {
                                    e.preventDefault();
                                    e.stopPropagation();
                                }}
                            >
                                <div className="financial-modal-header">
                                    <h2>Detalhes do Financeiro</h2>
                                    <button
                                        className="financial-modal-close"
                                        onClick={(e) => closeModal(e)}
                                    >
                                        ✕
                                    </button>
                                </div>

                                <div
                                    className="financial-modal-content"
                                    ref={scrollRef}
                                >
                                    {/* Contract Info */}
                                    <div className="financial-modal-section">
                                        <h3>
                                            {config.labels?.contract ||
                                                "Contrato"}
                                        </h3>
                                        <div className="financial-modal-info-grid">
                                            <div className="financial-modal-info-item">
                                                <label>
                                                    {config.labels?.client ||
                                                        "Cliente"}
                                                </label>
                                                <span>
                                                    {
                                                        getContractInfo(
                                                            selectedFinancial,
                                                        ).clientName
                                                    }
                                                </span>
                                            </div>
                                            <div className="financial-modal-info-item">
                                                <label>
                                                    {config.labels?.contract ||
                                                        "Contrato"}
                                                </label>
                                                <span>
                                                    {
                                                        getContractInfo(
                                                            selectedFinancial,
                                                        ).model
                                                    }
                                                </span>
                                            </div>
                                        </div>
                                    </div>

                                    {/* Financial Info */}
                                    <div className="financial-modal-section">
                                        <h3>Modelo de Financeiro</h3>
                                        <div className="financial-modal-info-grid">
                                            <div className="financial-modal-info-item">
                                                <label>Tipo</label>
                                                <span>
                                                    {financialApi.getFinancialTypeLabel(
                                                        selectedFinancial.financial_type,
                                                    )}
                                                </span>
                                            </div>
                                            {selectedFinancial.recurrence_type && (
                                                <div className="financial-modal-info-item">
                                                    <label>Recorrência</label>
                                                    <span>
                                                        {financialApi.getRecurrenceTypeLabel(
                                                            selectedFinancial.recurrence_type,
                                                        )}
                                                    </span>
                                                </div>
                                            )}
                                            {selectedFinancial.due_day && (
                                                <div className="financial-modal-info-item">
                                                    <label>
                                                        Dia de Vencimento
                                                    </label>
                                                    <span>
                                                        Dia{" "}
                                                        {
                                                            selectedFinancial.due_day
                                                        }
                                                    </span>
                                                </div>
                                            )}
                                            {selectedFinancial.description && (
                                                <div className="financial-modal-info-item financial-modal-info-full">
                                                    <label>Descrição</label>
                                                    <span>
                                                        {
                                                            selectedFinancial.description
                                                        }
                                                    </span>
                                                </div>
                                            )}
                                        </div>
                                    </div>

                                    {/* Values */}
                                    <div className="financial-modal-section">
                                        <h3>Valores</h3>
                                        <div className="financial-modal-values">
                                            <div className="financial-modal-value-card">
                                                <div className="financial-modal-value-label">
                                                    {config.labels?.client ||
                                                        "Cliente"}{" "}
                                                    Paga
                                                </div>
                                                <div className="financial-modal-value-amount">
                                                    {financialApi.formatCurrency(
                                                        selectedFinancial.total_client_value ||
                                                        selectedFinancial.client_value,
                                                    )}
                                                </div>
                                                {selectedFinancial.financial_type ===
                                                    "personalizado" && (
                                                        <div className="financial-modal-value-paid">
                                                            {config.labels
                                                                ?.client ||
                                                                "Cliente"}{" "}
                                                            pagou{" "}
                                                            {financialApi.formatCurrency(
                                                                paidClientTotal,
                                                            )}
                                                        </div>
                                                    )}
                                            </div>
                                            <div className="financial-modal-value-card financial-modal-value-received">
                                                <div className="financial-modal-value-label">
                                                    Você Recebe
                                                </div>
                                                <div className="financial-modal-value-amount">
                                                    {financialApi.formatCurrency(
                                                        selectedFinancial.total_received_value ||
                                                        selectedFinancial.received_value,
                                                    )}
                                                </div>
                                                {selectedFinancial.financial_type ===
                                                    "personalizado" && (
                                                        <div className="financial-modal-value-paid">
                                                            Você recebeu{" "}
                                                            {financialApi.formatCurrency(
                                                                paidReceivedTotal,
                                                            )}
                                                        </div>
                                                    )}
                                            </div>
                                        </div>
                                    </div>

                                    {/* Installments */}
                                    {selectedInstallments.length > 0 && (
                                        <div className="financial-modal-section">
                                            <h3>
                                                Parcelas (
                                                {selectedFinancial.paid_installments ||
                                                    0}
                                                /
                                                {
                                                    selectedFinancial.total_installments
                                                }{" "}
                                                pagas)
                                            </h3>
                                            <div className="financial-modal-installments">
                                                {selectedInstallments.map(
                                                    (inst) => {
                                                        const statusInfo =
                                                            financialApi.getInstallmentStatusInfo(
                                                                inst.status,
                                                            );
                                                        return (
                                                            <div
                                                                key={inst.id}
                                                                className={`financial-modal-installment financial-modal-installment-${inst.status}`}
                                                            >
                                                                <div className="financial-modal-installment-info">
                                                                    <span className="financial-modal-installment-label">
                                                                        {inst.installment_label ||
                                                                            financialApi.getInstallmentLabel(
                                                                                inst.installment_number,
                                                                            )}
                                                                    </span>
                                                                    <span
                                                                        className="financial-modal-installment-status"
                                                                        style={{
                                                                            color: statusInfo.color,
                                                                        }}
                                                                    >
                                                                        {
                                                                            statusInfo.label
                                                                        }
                                                                    </span>
                                                                </div>
                                                                <div className="financial-modal-installment-values">
                                                                    <span>
                                                                        {config
                                                                            .labels
                                                                            ?.client ||
                                                                            "Cliente"}
                                                                        :{" "}
                                                                        {financialApi.formatCurrency(
                                                                            inst.client_value,
                                                                        )}
                                                                    </span>
                                                                    <span>
                                                                        Recebido:{" "}
                                                                        {financialApi.formatCurrency(
                                                                            inst.received_value,
                                                                        )}
                                                                    </span>
                                                                </div>
                                                                {inst.due_date && (
                                                                    <div className="financial-modal-installment-date">
                                                                        Vence:{" "}
                                                                        {new Date(
                                                                            inst.due_date,
                                                                        ).toLocaleDateString(
                                                                            "pt-BR",
                                                                        )}
                                                                    </div>
                                                                )}
                                                                <div className="financial-modal-installment-actions">
                                                                    {inst.status ===
                                                                        "pago" ? (
                                                                        <button
                                                                            className="financial-modal-installment-btn financial-modal-installment-unpay"
                                                                            onClick={(
                                                                                e,
                                                                            ) =>
                                                                                handleMarkPending(
                                                                                    inst.id,
                                                                                    e,
                                                                                )
                                                                            }
                                                                        >
                                                                            Desfazer
                                                                        </button>
                                                                    ) : (
                                                                        <button
                                                                            className="financial-modal-installment-btn financial-modal-installment-pay"
                                                                            onClick={(
                                                                                e,
                                                                            ) =>
                                                                                handleMarkPaid(
                                                                                    inst.id,
                                                                                    null,
                                                                                    e,
                                                                                )
                                                                            }
                                                                        >
                                                                            Marcar
                                                                            Pago
                                                                        </button>
                                                                    )}
                                                                </div>
                                                            </div>
                                                        );
                                                    },
                                                )}
                                            </div>
                                        </div>
                                    )}
                                </div>

                                <div className="financial-modal-footer">
                                    <button
                                        className="financial-modal-btn-close"
                                        onClick={(e) => closeModal(e)}
                                    >
                                        Fechar
                                    </button>
                                </div>
                            </div>
                        </div>
                    );
                })()}
        </div>
    );
}
