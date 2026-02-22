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

import React, { useState, useEffect, useCallback, useRef } from "react";
import { useConfig } from "../contexts/ConfigContext";
import { useData } from "../contexts/DataContext";
import "./styles/Dashboard.css";

export default function Dashboard({ token, apiUrl, onTokenExpired }) {
    const { config, getPersistentFilter, setPersistentFilter } = useConfig();
    const { fetchContracts, fetchClients, fetchCategories, fetchWithCache } =
        useData();
    const [contracts, setContracts] = useState([]);
    const [clients, setClients] = useState([]);
    const [categories, setCategories] = useState([]);
    const [lines, setLines] = useState([]);
    const [loading, setLoading] = useState(true);
    const [dataLoading, setDataLoading] = useState(true);
    const [error, setError] = useState("");
    // Phase 1: Aggregated counts from /api/dashboard/counts (instant)
    const [dashCounts, setDashCounts] = useState(null);
    const [birthdayFilter, setBirthdayFilter] = useState(() =>
        getPersistentFilter("dashboard_birthday_filter", "day"),
    );

    // Tab state for info section (birthdays, expiring, expired)
    const [activeInfoTab, setActiveInfoTab] = useState(() =>
        getPersistentFilter("dashboard_active_info_tab", "birthdays"),
    );

    // Pagination state for each tab
    const [birthdayPage, setBirthdayPage] = useState(1);
    const [expiringPage, setExpiringPage] = useState(1);
    const [expiredPage, setExpiredPage] = useState(1);
    const itemsPerPage = 10;

    // Dashboard settings from backend
    const [dashboardSettings, setDashboardSettings] = useState({
        show_birthdays: true,
        birthdays_days_ahead: 7,
        show_recent_activity: true,
        recent_activity_count: 15,
        show_statistics: true,
        show_expiring_contracts: true,
        show_expired_contracts: true,
        expiring_days_ahead: 30,
        show_quick_actions: true,
    });

    // Save persistent states
    useEffect(() => {
        if (birthdayFilter) {
            setPersistentFilter("dashboard_birthday_filter", birthdayFilter);
        }
    }, [birthdayFilter, setPersistentFilter]);

    useEffect(() => {
        setPersistentFilter("dashboard_active_info_tab", activeInfoTab);
    }, [activeInfoTab, setPersistentFilter]);

    // Load dashboard settings from backend
    // StrictMode-safe guard: prevent double-firing
    const dashSettingsLoaded = useRef(false);
    useEffect(() => {
        if (dashSettingsLoaded.current) return;
        const loadDashboardSettings = async () => {
            try {
                const response = await fetch(
                    `${apiUrl}/system-config/dashboard`,
                    {
                        headers: {
                            Authorization: `Bearer ${token}`,
                        },
                    },
                );
                if (response.ok) {
                    const data = await response.json();
                    setDashboardSettings((prev) => ({ ...prev, ...data }));
                }
            } catch (err) {
                console.error("Error loading dashboard settings:", err);
            }
        };
        if (token && apiUrl) {
            dashSettingsLoaded.current = true;
            loadDashboardSettings();
        }
    }, [token, apiUrl]);

    // Two-phase loading:
    // Phase 1: Load counts instantly (single lightweight query)
    // Phase 2: Load detailed data in background for tabs (birthdays, expiring, etc.)
    // StrictMode-safe guard: prevent double-firing of initial load
    const dataLoadStarted = useRef(false);
    useEffect(() => {
        if (dataLoadStarted.current) return;
        dataLoadStarted.current = true;
        loadCounts();
        loadData();
    }, []);

    const loadCounts = useCallback(async () => {
        try {
            const expiringDays = dashboardSettings.expiring_days_ahead || 30;
            const response = await fetchWithCache(
                "/dashboard/counts",
                { params: { expiring_days: expiringDays } },
                "dashboard_counts",
                30 * 1000,
                true,
            );
            if (response?.data) {
                setDashCounts(response.data);
            }
            // Phase 1 complete — stats cards can render
            setLoading(false);
        } catch (err) {
            console.warn("Failed to load dashboard counts:", err);
            // Fall through to Phase 2 which will also provide counts
            setLoading(false);
        }
    }, [fetchWithCache, dashboardSettings.expiring_days_ahead]);

    const loadData = useCallback(async () => {
        setDataLoading(true);
        setError("");

        try {
            // Phase 2: Fetch full data for tabs (birthdays, expiring, recent activity)
            // Use cache (forceRefresh=false) since this data may already be
            // loaded by Contracts, Financial, or Clients pages.
            const [contractsData, clientsData, categoriesData] =
                await Promise.all([
                    fetchContracts({}, false),
                    fetchClients({}, false),
                    fetchCategories(false),
                ]);

            setContracts(contractsData.data || []);
            setClients(clientsData.data || []);
            setCategories(categoriesData.data || []);

            // Use lines already returned by the categories endpoint
            const flattenedLines = (categoriesData.data || []).flatMap(
                (category) => category.lines || [],
            );
            setLines(flattenedLines);

            // Always set birthday filter to 'day'
            if (!birthdayFilter) {
                setBirthdayFilter("day");
            }
        } catch (err) {
            setError(err.message);
        } finally {
            setDataLoading(false);
        }
    }, [fetchContracts, fetchClients, fetchCategories, birthdayFilter]);

    const getContractStatus = (contract) => {
        // Se não tem data de término, é considerado ativo
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
        } else if (
            daysUntilExpiration <= dashboardSettings.expiring_days_ahead
        ) {
            return { status: "Expirando", color: "#f39c12" };
        } else {
            return { status: "Ativo", color: "#27ae60" };
        }
    };

    const notStartedContracts = contracts.filter((c) => {
        const status = getContractStatus(c);
        return !c.archived_at && status.status === "Não Iniciado";
    });

    const activeContracts = contracts.filter((c) => {
        const status = getContractStatus(c);
        return !c.archived_at && status.status === "Ativo";
    });

    const expiringContracts = contracts
        .filter((c) => {
            const status = getContractStatus(c);
            return !c.archived_at && status.status === "Expirando";
        })
        .sort((a, b) => {
            // Sort by days remaining (ascending - fewer days first)
            const endDateA = new Date(a.end_date);
            const endDateB = new Date(b.end_date);
            const now = new Date();
            const daysLeftA = Math.ceil(
                (endDateA - now) / (1000 * 60 * 60 * 24),
            );
            const daysLeftB = Math.ceil(
                (endDateB - now) / (1000 * 60 * 60 * 24),
            );
            return daysLeftA - daysLeftB;
        });

    const expiredContracts = contracts
        .filter((c) => {
            const status = getContractStatus(c);
            return !c.archived_at && status.status === "Expirado";
        })
        .sort((a, b) => {
            // Sort by days expired (descending - more days expired first)
            const endDateA = new Date(a.end_date);
            const endDateB = new Date(b.end_date);
            const now = new Date();
            const daysExpiredA = Math.ceil(
                (now - endDateA) / (1000 * 60 * 60 * 24),
            );
            const daysExpiredB = Math.ceil(
                (now - endDateB) / (1000 * 60 * 60 * 24),
            );
            return daysExpiredB - daysExpiredA;
        });

    const activeClients = clients.filter(
        (c) => !c.archived_at && c.status === "ativo",
    );

    const inactiveClients = clients.filter(
        (c) => !c.archived_at && c.status === "inativo",
    );

    const archivedClients = clients.filter((c) => c.archived_at);

    const totalCategories = categories.length;
    const totalLines = lines.length;

    // Use dashCounts for instant stats, fall back to computed values once full data loads
    const statsActiveClients =
        dashCounts?.clients?.active ?? activeClients.length;
    const statsInactiveClients =
        dashCounts?.clients?.inactive ?? inactiveClients.length;
    const statsArchivedClients =
        dashCounts?.clients?.archived ?? archivedClients.length;
    const statsActiveContracts =
        dashCounts?.contracts?.active ?? activeContracts.length;
    const statsExpiringContracts =
        dashCounts?.contracts?.expiring ?? expiringContracts.length;
    const statsExpiredContracts =
        dashCounts?.contracts?.expired ?? expiredContracts.length;
    const statsNotStartedContracts =
        dashCounts?.contracts?.not_started ?? notStartedContracts.length;
    const statsTotalCategories =
        dashCounts?.categories?.total ?? totalCategories;
    const statsTotalLines = dashCounts?.categories?.subcategories ?? totalLines;

    // Clientes que fazem aniversário no mês atual ou dia atual (respeitando days_ahead)
    const birthdayClients = clients
        .filter((c) => {
            if (c.archived_at || !c.birth_date) {
                return false;
            }

            // Parse birth_date as UTC to avoid timezone issues
            const birthDateStr = c.birth_date.split("T")[0]; // Get YYYY-MM-DD part
            const [birthYear, birthMonth, birthDay] = birthDateStr
                .split("-")
                .map(Number);

            const now = new Date();
            now.setHours(0, 0, 0, 0);
            const currentMonth = now.getMonth() + 1; // getMonth returns 0-11, so add 1
            const currentDay = now.getDate();

            if (birthdayFilter === "day") {
                return birthMonth === currentMonth && birthDay === currentDay;
            } else if (birthdayFilter === "week") {
                // Check if birthday is within the next X days
                const daysAhead = dashboardSettings.birthdays_days_ahead || 7;
                const today = new Date();
                today.setHours(0, 0, 0, 0);

                for (let i = 0; i <= daysAhead; i++) {
                    const checkDate = new Date(today);
                    checkDate.setDate(checkDate.getDate() + i);
                    const checkMonth = checkDate.getMonth() + 1; // Convert 0-11 to 1-12
                    const checkDay = checkDate.getDate();
                    if (checkMonth === birthMonth && checkDay === birthDay) {
                        return true;
                    }
                }
                return false;
            } else {
                // Default to month if birthdayFilter is null or 'month'
                return birthMonth === currentMonth;
            }
        })
        .sort((a, b) => {
            // Sort by birth month and day (ascending)
            const aDateStr = a.birth_date.split("T")[0];
            const [aYear, aMonth, aDay] = aDateStr.split("-").map(Number);
            const bDateStr = b.birth_date.split("T")[0];
            const [bYear, bMonth, bDay] = bDateStr.split("-").map(Number);

            if (aMonth !== bMonth) {
                return aMonth - bMonth;
            }
            if (aDay !== bDay) {
                return aDay - bDay;
            }
            // If same date, sort alphabetically by name
            return a.name.localeCompare(b.name);
        });

    // Data subsets for recent activities
    const archivedClientsList = clients.filter((c) => c.archived_at);
    const archivedContractsList = contracts.filter((c) => c.archived_at);

    // Client Suggestions: Non-archived clients older than 30 days without active contracts (none or all expired)
    const now = new Date();
    const clientSuggestions = clients
        .filter((c) => {
            if (c.archived_at) return false;

            const createdDate = new Date(c.created_at || c.updated_at);
            const daysOld = (now - createdDate) / (1000 * 60 * 60 * 24);
            if (daysOld < 30) return false;

            const clientContracts = contracts.filter(
                (con) => con.client_id === c.id,
            );
            if (clientContracts.length === 0) return true;

            const allInactive = clientContracts.every((con) => {
                const endDate = new Date(con.end_date);
                return endDate < now || con.archived_at;
            });

            return allInactive;
        })
        .map((c) => ({
            type: "client_suggestion",
            date: new Date(now.getTime() - 1000), // Slightly in the past so it stays below fresh alerts
            label: `💡 Sugestão: Retomar ${config.labels?.client || "Cliente"} ${c.name} (Sem contrato ativo)`,
            id: `sug_${c.id}`,
        }));

    // Recent Activities calculation
    const recentActivities = [
        ...clients
            .filter((c) => !c.archived_at)
            .map((c) => ({
                type: "new_client",
                date: new Date(c.created_at || c.updated_at),
                label: `Novo ${config.labels?.client || "Cliente"}: ${c.name}`,
                id: `nc_${c.id}`,
            })),
        ...contracts
            .filter((c) => !c.archived_at)
            .map((c) => ({
                type: "new_contract",
                date: new Date(c.created_at || c.updated_at),
                label: `Novo ${config.labels?.contract || "Contrato"}: ${c.model || "Sem modelo"}`,
                id: `nct_${c.id}`,
            })),
        ...archivedClientsList.map((c) => ({
            type: "archived_client",
            date: new Date(c.archived_at),
            label: `${config.labels?.client || "Cliente"} Arquivado: ${c.name}`,
            id: `ac_${c.id}`,
        })),
        ...archivedContractsList.map((c) => ({
            type: "archived_contract",
            date: new Date(c.archived_at),
            label: `${config.labels?.contract || "Contrato"} Arquivado: ${c.model || "Sem modelo"}`,
            id: `act_${c.id}`,
        })),
        ...clientSuggestions,
    ]
        .filter((a) => !isNaN(a.date.getTime()))
        .sort((a, b) => b.date - a.date)
        .slice(0, dashboardSettings.recent_activity_count || 15);

    if (loading) {
        return (
            <div className="dashboard-loading">
                <div className="dashboard-loading-text">Carregando...</div>
            </div>
        );
    }

    if (error) {
        return (
            <div className="dashboard-error-container">
                <div className="dashboard-error-message">{error}</div>
                <button onClick={loadData} className="dashboard-error-button">
                    Tentar Novamente
                </button>
            </div>
        );
    }

    return (
        <div>
            <div
                style={{
                    display: "flex",
                    justifyContent: "space-between",
                    alignItems: "center",
                    marginBottom: "20px",
                }}
            >
                <h1 className="dashboard-title" style={{ margin: 0 }}>
                    📈 Dashboard
                </h1>
            </div>

            {/* Statistics Cards - controlled by show_statistics */}
            {dashboardSettings.show_statistics && (
                <div className="dashboard-stats-grid">
                    <div className="dashboard-stat-card">
                        <div className="dashboard-stat-label">
                            {config.labels.clients} Ativos
                        </div>
                        <div className="dashboard-stat-value clients">
                            {statsActiveClients}
                        </div>
                    </div>

                    <div className="dashboard-stat-card">
                        <div className="dashboard-stat-label">
                            {config.labels.contracts || "Contratos"} Ativos
                        </div>
                        <div className="dashboard-stat-value active">
                            {statsActiveContracts}
                        </div>
                    </div>

                    <div className="dashboard-stat-card">
                        <div className="dashboard-stat-label">
                            Expirando em Breve
                        </div>
                        <div className="dashboard-stat-value expiring">
                            {statsExpiringContracts}
                        </div>
                    </div>

                    <div className="dashboard-stat-card">
                        <div className="dashboard-stat-label">
                            {config.labels.contracts || "Contratos"} Expirados
                        </div>
                        <div className="dashboard-stat-value expired">
                            {statsExpiredContracts}
                        </div>
                    </div>

                    <div className="dashboard-stat-card">
                        <div className="dashboard-stat-label">
                            {config.labels.clients} Inativos
                        </div>
                        <div className="dashboard-stat-value inactive">
                            {statsInactiveClients}
                        </div>
                    </div>

                    <div className="dashboard-stat-card">
                        <div className="dashboard-stat-label">
                            {config.labels.clients} Arquivados
                        </div>
                        <div className="dashboard-stat-value archived">
                            {statsArchivedClients}
                        </div>
                    </div>

                    <div className="dashboard-stat-card">
                        <div className="dashboard-stat-label">
                            {config.labels.categories}
                        </div>
                        <div className="dashboard-stat-value categories">
                            {statsTotalCategories}
                        </div>
                    </div>

                    <div className="dashboard-stat-card">
                        <div className="dashboard-stat-label">
                            {config.labels.subcategories}
                        </div>
                        <div className="dashboard-stat-value lines">
                            {statsTotalLines}
                        </div>
                    </div>
                </div>
            )}

            {/* 1. Quick Actions Section - Single Row */}
            {dashboardSettings.show_quick_actions && (
                <div
                    className="dashboard-section-card"
                    style={{ marginTop: "20px" }}
                >
                    <h2 className="dashboard-section-title">
                        ⚡ Ações Rápidas
                    </h2>
                    <div className="dashboard-quick-actions">
                        <button
                            className="dashboard-action-btn"
                            onClick={() =>
                            (window.location.hash =
                                "#/contracts?action=new")
                            }
                        >
                            <span className="action-icon">📄</span>
                            <span className="action-label">
                                Novo {config.labels?.contract || "Contrato"}
                            </span>
                        </button>
                        <button
                            className="dashboard-action-btn"
                            onClick={() =>
                                (window.location.hash = "#/clients?action=new")
                            }
                        >
                            <span className="action-icon">👤</span>
                            <span className="action-label">
                                Novo {config.labels?.client || "Cliente"}
                            </span>
                        </button>
                        <button className="dashboard-action-btn">
                            <span className="action-icon">🤝</span>
                            <span className="action-label">
                                Novo {config.labels?.affiliate || "Afiliado"}
                            </span>
                        </button>
                        <button
                            className="dashboard-action-btn"
                            onClick={() =>
                            (window.location.hash =
                                "#/categories?action=new")
                            }
                        >
                            <span className="action-icon">🏷️</span>
                            <span className="action-label">
                                Nova {config.labels?.category || "Categoria"}
                            </span>
                        </button>
                        <button className="dashboard-action-btn">
                            <span className="action-icon">📂</span>
                            <span className="action-label">
                                Nova{" "}
                                {config.labels?.subcategory || "Subcategoria"}
                            </span>
                        </button>
                    </div>
                </div>
            )}

            {/* 2. Tabbed Info Section - Birthdays, Expiring, Expired */}
            <div
                className="dashboard-section-card"
                style={{ marginTop: "20px" }}
            >
                <h2 className="dashboard-section-title">Entre em contato:</h2>
                <div className="dashboard-info-tabs">
                    <button
                        className={`dashboard-info-tab ${activeInfoTab === "birthdays" ? "active" : ""}`}
                        onClick={() => setActiveInfoTab("birthdays")}
                    >
                        🎂 Aniversariantes ({birthdayClients.length})
                    </button>
                    <button
                        className={`dashboard-info-tab ${activeInfoTab === "expiring" ? "active" : ""}`}
                        onClick={() => setActiveInfoTab("expiring")}
                    >
                        ⌛ Expirando ({statsExpiringContracts})
                    </button>
                    <button
                        className={`dashboard-info-tab ${activeInfoTab === "expired" ? "active" : ""}`}
                        onClick={() => setActiveInfoTab("expired")}
                    >
                        ❌ Expirados ({statsExpiredContracts})
                    </button>
                </div>

                <div className="dashboard-info-content">
                    {/* Birthdays Tab Content */}
                    {activeInfoTab === "birthdays" && (
                        <>
                            <div
                                style={{
                                    marginBottom: "15px",
                                    display: "flex",
                                    gap: "8px",
                                }}
                            >
                                <button
                                    onClick={() => {
                                        setBirthdayFilter("day");
                                        setBirthdayPage(1);
                                    }}
                                    className={`filter-btn ${birthdayFilter === "day" ? "active" : ""}`}
                                >
                                    Hoje
                                </button>
                                <button
                                    onClick={() => {
                                        setBirthdayFilter("week");
                                        setBirthdayPage(1);
                                    }}
                                    className={`filter-btn ${birthdayFilter === "week" ? "active" : ""}`}
                                >
                                    Próxs.{" "}
                                    {dashboardSettings.birthdays_days_ahead}{" "}
                                    dias
                                </button>
                                <button
                                    onClick={() => {
                                        setBirthdayFilter("month");
                                        setBirthdayPage(1);
                                    }}
                                    className={`filter-btn ${birthdayFilter === "month" ? "active" : ""}`}
                                >
                                    Mês
                                </button>
                            </div>
                            {birthdayClients.length === 0 ? (
                                <p className="dashboard-section-empty">
                                    Nenhum aniversariante
                                </p>
                            ) : (
                                <>
                                    <div className="dashboard-contracts-list">
                                        {birthdayClients
                                            .slice(
                                                (birthdayPage - 1) *
                                                itemsPerPage,
                                                birthdayPage * itemsPerPage,
                                            )
                                            .map((client) => {
                                                // Find last contract for this client
                                                const clientContracts =
                                                    contracts.filter(
                                                        (c) =>
                                                            c.client_id ===
                                                            client.id &&
                                                            !c.archived_at,
                                                    );
                                                const lastContract =
                                                    clientContracts.length > 0
                                                        ? clientContracts.sort(
                                                            (a, b) =>
                                                                new Date(
                                                                    b.end_date ||
                                                                    0,
                                                                ) -
                                                                new Date(
                                                                    a.end_date ||
                                                                    0,
                                                                ),
                                                        )[0]
                                                        : null;

                                                // Format birth date
                                                const dateStr =
                                                    client.birth_date;
                                                let formattedDate = "-";
                                                if (
                                                    dateStr &&
                                                    dateStr.match(
                                                        /^\d{4}-\d{2}-\d{2}/,
                                                    )
                                                ) {
                                                    const [year, month, day] =
                                                        dateStr
                                                            .split("T")[0]
                                                            .split("-");
                                                    formattedDate = `${day}/${month}`;
                                                }

                                                return (
                                                    <div
                                                        key={client.id}
                                                        className="dashboard-contract-item clients single-line"
                                                    >
                                                        <div className="dashboard-contract-date">
                                                            {formattedDate}
                                                        </div>
                                                        <div className="dashboard-contract-model">
                                                            {client.name}
                                                        </div>
                                                        <div className="dashboard-contract-phone">
                                                            {client.phone ||
                                                                "-"}
                                                        </div>
                                                        <div className="dashboard-contract-key">
                                                            {lastContract?.model ||
                                                                "Sem contrato"}
                                                        </div>
                                                    </div>
                                                );
                                            })}
                                    </div>
                                    {birthdayClients.length > itemsPerPage && (
                                        <div className="dashboard-pagination">
                                            <button
                                                className="pagination-btn"
                                                disabled={birthdayPage === 1}
                                                onClick={() =>
                                                    setBirthdayPage(
                                                        (p) => p - 1,
                                                    )
                                                }
                                            >
                                                ← Anterior
                                            </button>
                                            <span className="pagination-info">
                                                Página {birthdayPage} de{" "}
                                                {Math.ceil(
                                                    birthdayClients.length /
                                                    itemsPerPage,
                                                )}
                                            </span>
                                            <button
                                                className="pagination-btn"
                                                disabled={
                                                    birthdayPage >=
                                                    Math.ceil(
                                                        birthdayClients.length /
                                                        itemsPerPage,
                                                    )
                                                }
                                                onClick={() =>
                                                    setBirthdayPage(
                                                        (p) => p + 1,
                                                    )
                                                }
                                            >
                                                Próxima →
                                            </button>
                                        </div>
                                    )}
                                </>
                            )}
                        </>
                    )}

                    {/* Expiring Tab Content */}
                    {activeInfoTab === "expiring" && (
                        <>
                            {expiringContracts.length === 0 ? (
                                <p className="dashboard-section-empty">
                                    Nenhum{" "}
                                    {config.labels?.contract?.toLowerCase() ||
                                        "contrato"}{" "}
                                    expirando
                                </p>
                            ) : (
                                <>
                                    <div className="dashboard-contracts-list">
                                        {expiringContracts
                                            .slice(
                                                (expiringPage - 1) *
                                                itemsPerPage,
                                                expiringPage * itemsPerPage,
                                            )
                                            .map((contract) => {
                                                const endDate = new Date(
                                                    contract.end_date,
                                                );
                                                const now = new Date();
                                                const daysLeft = Math.ceil(
                                                    (endDate - now) /
                                                    (1000 * 60 * 60 * 24),
                                                );
                                                // Find client for this contract
                                                const client = clients.find(
                                                    (c) =>
                                                        c.id ===
                                                        contract.client_id,
                                                );

                                                return (
                                                    <div
                                                        key={contract.id}
                                                        className="dashboard-contract-item expiring single-line"
                                                    >
                                                        <div className="dashboard-contract-client-name">
                                                            {client?.name ||
                                                                "Sem cliente"}
                                                        </div>
                                                        <div className="dashboard-contract-model">
                                                            {contract.model ||
                                                                "-"}
                                                        </div>
                                                        <div className="dashboard-contract-phone">
                                                            {client?.phone ||
                                                                "-"}
                                                        </div>
                                                        <div className="dashboard-contract-days expiring">
                                                            {daysLeft} dias
                                                        </div>
                                                    </div>
                                                );
                                            })}
                                    </div>
                                    {expiringContracts.length >
                                        itemsPerPage && (
                                            <div className="dashboard-pagination">
                                                <button
                                                    className="pagination-btn"
                                                    disabled={expiringPage === 1}
                                                    onClick={() =>
                                                        setExpiringPage(
                                                            (p) => p - 1,
                                                        )
                                                    }
                                                >
                                                    ← Anterior
                                                </button>
                                                <span className="pagination-info">
                                                    Página {expiringPage} de{" "}
                                                    {Math.ceil(
                                                        expiringContracts.length /
                                                        itemsPerPage,
                                                    )}
                                                </span>
                                                <button
                                                    className="pagination-btn"
                                                    disabled={
                                                        expiringPage >=
                                                        Math.ceil(
                                                            expiringContracts.length /
                                                            itemsPerPage,
                                                        )
                                                    }
                                                    onClick={() =>
                                                        setExpiringPage(
                                                            (p) => p + 1,
                                                        )
                                                    }
                                                >
                                                    Próxima →
                                                </button>
                                            </div>
                                        )}
                                </>
                            )}
                        </>
                    )}

                    {/* Expired Tab Content */}
                    {activeInfoTab === "expired" && (
                        <>
                            {expiredContracts.length === 0 ? (
                                <p className="dashboard-section-empty">
                                    Nenhum{" "}
                                    {config.labels?.contract?.toLowerCase() ||
                                        "contrato"}{" "}
                                    expirado
                                </p>
                            ) : (
                                <>
                                    <div className="dashboard-contracts-list">
                                        {expiredContracts
                                            .slice(
                                                (expiredPage - 1) *
                                                itemsPerPage,
                                                expiredPage * itemsPerPage,
                                            )
                                            .map((contract) => {
                                                const endDate = new Date(
                                                    contract.end_date,
                                                );
                                                const now = new Date();
                                                const daysExpired = Math.ceil(
                                                    (now - endDate) /
                                                    (1000 * 60 * 60 * 24),
                                                );
                                                // Find client for this contract
                                                const client = clients.find(
                                                    (c) =>
                                                        c.id ===
                                                        contract.client_id,
                                                );

                                                return (
                                                    <div
                                                        key={contract.id}
                                                        className="dashboard-contract-item expired single-line"
                                                    >
                                                        <div className="dashboard-contract-client-name">
                                                            {client?.name ||
                                                                "Sem cliente"}
                                                        </div>
                                                        <div className="dashboard-contract-model">
                                                            {contract.model ||
                                                                "-"}
                                                        </div>
                                                        <div className="dashboard-contract-phone">
                                                            {client?.phone ||
                                                                "-"}
                                                        </div>
                                                        <div className="dashboard-contract-days expired">
                                                            há {daysExpired}{" "}
                                                            dias
                                                        </div>
                                                    </div>
                                                );
                                            })}
                                    </div>
                                    {expiredContracts.length > itemsPerPage && (
                                        <div className="dashboard-pagination">
                                            <button
                                                className="pagination-btn"
                                                disabled={expiredPage === 1}
                                                onClick={() =>
                                                    setExpiredPage((p) => p - 1)
                                                }
                                            >
                                                ← Anterior
                                            </button>
                                            <span className="pagination-info">
                                                Página {expiredPage} de{" "}
                                                {Math.ceil(
                                                    expiredContracts.length /
                                                    itemsPerPage,
                                                )}
                                            </span>
                                            <button
                                                className="pagination-btn"
                                                disabled={
                                                    expiredPage >=
                                                    Math.ceil(
                                                        expiredContracts.length /
                                                        itemsPerPage,
                                                    )
                                                }
                                                onClick={() =>
                                                    setExpiredPage((p) => p + 1)
                                                }
                                            >
                                                Próxima →
                                            </button>
                                        </div>
                                    )}
                                </>
                            )}
                        </>
                    )}
                </div>
            </div>

            {/* 3. Recent Activities Section */}
            {dashboardSettings.show_recent_activity && (
                <div
                    className="dashboard-section-card"
                    style={{ marginTop: "20px" }}
                >
                    <h2 className="dashboard-section-title">
                        📋 Atividade Recente
                    </h2>
                    <div className="dashboard-activities-list">
                        {recentActivities.length === 0 ? (
                            <p className="dashboard-section-empty">
                                Nenhuma atividade recente
                            </p>
                        ) : (
                            recentActivities.map((activity) => (
                                <div
                                    key={activity.id}
                                    className="dashboard-activity-item"
                                >
                                    <div
                                        className="activity-dot"
                                        data-type={activity.type}
                                    ></div>
                                    <div className="activity-content">
                                        <div className="activity-label">
                                            {activity.label}
                                        </div>
                                        <div className="activity-date">
                                            {activity.date.toLocaleString(
                                                "pt-BR",
                                                {
                                                    day: "2-digit",
                                                    month: "2-digit",
                                                    hour: "2-digit",
                                                    minute: "2-digit",
                                                },
                                            )}
                                        </div>
                                    </div>
                                </div>
                            ))
                        )}
                    </div>
                </div>
            )}
        </div>
    );
}
