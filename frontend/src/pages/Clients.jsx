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

import React, { useState, useEffect, useRef, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { useConfig } from "../contexts/ConfigContext";
import { useData } from "../contexts/DataContext";
import { clientsApi } from "../api/clientsApi";
import { useUrlState } from "../hooks/useUrlState";
import {
    formatClientForEdit,
    formatAffiliateForEdit,
    getInitialFormData,
    getInitialAffiliateForm,
    prepareClientPayload,
} from "../utils/clientHelpers";
import ClientModal from "../components/clients/ClientModal";
import ClientsTable from "../components/clients/ClientsTable";
import AffiliatesPanel from "../components/clients/AffiliatesPanel";
import AffiliateModal from "../components/clients/AffiliateModal";
import PrimaryButton from "../components/common/PrimaryButton";
import Pagination from "../components/common/Pagination";
import "./styles/Clients.css";

export default function Clients({ token, apiUrl, onTokenExpired }) {
    const navigate = useNavigate();
    const {
        fetchClients,
        fetchWithCache,
        createClient: createClientAPI,
        updateClient: updateClientAPI,
        deleteClient: deleteClientAPI,
        invalidateCache,
    } = useData();
    const [clients, setClients] = useState([]);
    const [totalItems, setTotalItems] = useState(0);
    const [counts, setCounts] = useState({
        total: 0,
        active: 0,
        inactive: 0,
        archived: 0,
    });
    const [loading, setLoading] = useState(true);
    const [tableLoading, setTableLoading] = useState(false);
    const [error, setError] = useState("");
    const [modalError, setModalError] = useState("");
    const [affiliateModalError, setAffiliateModalError] = useState("");
    const { config, getGenderHelpers } = useConfig();
    const g = getGenderHelpers("client");
    const filtersContainerRef = useRef(null);

    // State persistence
    const { values, updateValue, updateValues, updateValuesImmediate } =
        useUrlState(
            { filter: "active", search: "", page: "1", limit: "20" },
            { debounce: true, debounceTime: 300, syncWithUrl: false },
        );
    const filter = values.filter;
    const searchTerm = values.search;
    const currentPage = parseInt(values.page || "1", 10);
    const itemsPerPage = parseInt(values.limit || "20", 10);
    const setFilter = (val) => {
        updateValuesImmediate({ filter: val, page: "1" });
    };
    const setSearchTerm = (val) => {
        updateValues({ search: val, page: "1" });
    };
    const setCurrentPage = (page) =>
        updateValuesImmediate({ page: page.toString() });
    const setItemsPerPage = (limit) => {
        updateValuesImmediate({ limit: limit.toString(), page: "1" });
    };

    const [selectedClient, setSelectedClient] = useState(null);
    const [showModal, setShowModal] = useState(false);
    const [modalMode, setModalMode] = useState("create");
    const [showAffiliatesPanel, setShowAffiliatesPanel] = useState(false);
    const [showAffiliateModal, setShowAffiliateModal] = useState(false);
    const [affiliateModalMode, setAffiliateModalMode] = useState("create");
    const [affiliates, setAffiliates] = useState([]);
    const [selectedAffiliate, setSelectedAffiliate] = useState(null);
    const [formData, setFormData] = useState(getInitialFormData());
    const [affiliateForm, setAffiliateForm] = useState(
        getInitialAffiliateForm(),
    );

    // Initial load: counts + first page of data
    useEffect(() => {
        loadCounts();
        loadClients();
    }, []);

    // Reload data when filter/search/page/limit changes (skip initial mount)
    const initialLoadDone = useRef(false);
    useEffect(() => {
        if (!initialLoadDone.current) {
            initialLoadDone.current = true;
            return;
        }
        loadClients();
    }, [filter, searchTerm, currentPage, itemsPerPage]);

    // Equalize filter button widths
    useEffect(() => {
        if (filtersContainerRef.current) {
            const buttons = filtersContainerRef.current.querySelectorAll(
                ".clients-filter-button",
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
    }, [filter, clients]); // Re-calculate when filter or clients change

    const loadCounts = useCallback(async () => {
        try {
            const response = await fetchWithCache(
                "/clients/counts",
                {},
                "clients_counts",
                30 * 1000,
                false,
            );
            if (response?.data) {
                setCounts(response.data);
            }
        } catch (err) {
            console.warn("Failed to load client counts:", err);
        }
    }, [fetchWithCache]);

    const loadClients = useCallback(async () => {
        // Use tableLoading for subsequent loads (not full-page loading spinner)
        if (initialLoadDone.current) {
            setTableLoading(true);
        } else {
            setLoading(true);
        }
        setError("");
        try {
            const offset = (currentPage - 1) * itemsPerPage;
            const params = {
                include_stats: true,
                include_archived: true,
                limit: itemsPerPage,
                offset: offset,
            };
            // Only send filter if it's not "all" (backend returns all when no filter)
            if (filter && filter !== "all") {
                params.filter = filter;
            }
            if (searchTerm) {
                params.search = searchTerm;
            }
            const response = await fetchWithCache(
                "/clients",
                { params },
                null,
                0,
                true,
            );
            setClients(response?.data || []);
            setTotalItems(response?.total ?? (response?.data?.length || 0));
        } catch (err) {
            setError(err.message);
        } finally {
            setLoading(false);
            setTableLoading(false);
        }
    }, [fetchWithCache, filter, searchTerm, currentPage, itemsPerPage]);

    const loadAffiliates = async (clientId) => {
        try {
            const data = await clientsApi.loadAffiliates(
                apiUrl,
                token,
                clientId,
                onTokenExpired,
            );
            setAffiliates(data);
        } catch (err) {
            setError(err.message);
        }
    };

    const reloadAfterMutation = useCallback(async () => {
        invalidateCache("clients");
        await Promise.all([loadCounts(), loadClients()]);
    }, [invalidateCache, loadCounts, loadClients]);

    const createClient = async () => {
        setModalError("");
        try {
            const payload = prepareClientPayload(formData);
            await createClientAPI(payload);
            await reloadAfterMutation();
            closeModal();
        } catch (err) {
            setModalError(err.message);
        }
    };

    const updateClient = async () => {
        setModalError("");
        try {
            const payload = prepareClientPayload(formData);
            await updateClientAPI(selectedClient.id, payload);
            await reloadAfterMutation();
            closeModal();
        } catch (err) {
            setModalError(err.message);
        }
    };

    const archiveClient = async (clientId) => {
        if (!window.confirm("Tem certeza que deseja arquivar este cliente?"))
            return;

        try {
            await deleteClientAPI(clientId);
            await reloadAfterMutation();
        } catch (err) {
            setError(err.message);
        }
    };

    const unarchiveClient = async (clientId) => {
        try {
            // Note: We still need to use the API directly for unarchive
            // as DataContext doesn't have an unarchive method
            await clientsApi.unarchiveClient(
                apiUrl,
                token,
                clientId,
                onTokenExpired,
            );
            await reloadAfterMutation();
        } catch (err) {
            setError(err.message);
        }
    };

    const createAffiliate = async () => {
        setAffiliateModalError("");
        try {
            await clientsApi.createAffiliate(
                apiUrl,
                token,
                selectedClient.id,
                affiliateForm,
                onTokenExpired,
            );
            await loadAffiliates(selectedClient.id);
            closeAffiliateModal();
        } catch (err) {
            setAffiliateModalError(err.message);
        }
    };

    const updateAffiliate = async () => {
        setAffiliateModalError("");
        try {
            await clientsApi.updateAffiliate(
                apiUrl,
                token,
                selectedAffiliate.id,
                affiliateForm,
                onTokenExpired,
            );
            await loadAffiliates(selectedClient.id);
            closeAffiliateModal();
        } catch (err) {
            setAffiliateModalError(err.message);
        }
    };

    const deleteAffiliate = async (affiliateId) => {
        if (!window.confirm("Tem certeza que deseja deletar este afiliado?"))
            return;

        try {
            await clientsApi.deleteAffiliate(
                apiUrl,
                token,
                affiliateId,
                onTokenExpired,
            );
            await loadAffiliates(selectedClient.id);
        } catch (err) {
            setError(err.message);
        }
    };

    const openCreateModal = () => {
        setModalMode("create");
        setFormData(getInitialFormData());
        setShowModal(true);
    };

    const openEditModal = (client) => {
        setModalMode("edit");
        setSelectedClient(client);
        setFormData(formatClientForEdit(client));
        setShowModal(true);
    };

    const openAffiliatesPanel = async (client) => {
        setSelectedClient(client);
        await loadAffiliates(client.id);
        setShowAffiliatesPanel(true);
    };

    const closeAffiliatesPanel = () => {
        setShowAffiliatesPanel(false);
        setSelectedClient(null);
        setAffiliates([]);
    };

    const openCreateAffiliateModal = () => {
        setAffiliateModalMode("create");
        setSelectedAffiliate(null);
        setAffiliateForm(getInitialAffiliateForm());
        setShowAffiliateModal(true);
    };

    const openEditAffiliateModal = (affiliate) => {
        setAffiliateModalMode("edit");
        setSelectedAffiliate(affiliate);
        setAffiliateForm(formatAffiliateForEdit(affiliate));
        setShowAffiliateModal(true);
    };

    const closeAffiliateModal = () => {
        setShowAffiliateModal(false);
        setSelectedAffiliate(null);
        setAffiliateForm(getInitialAffiliateForm());
        setAffiliateModalError("");
    };

    const closeModal = () => {
        setShowModal(false);
        setSelectedClient(null);
        setModalError("");
    };

    const handleSubmit = (e) => {
        e.preventDefault();
        if (modalMode === "create") {
            createClient();
        } else {
            updateClient();
        }
    };

    const handleAffiliateSubmit = (e) => {
        e.preventDefault();
        if (affiliateModalMode === "create") {
            createAffiliate();
        } else {
            updateAffiliate();
        }
    };

    const onViewContracts = (client) => {
        navigate(`/contracts?clientName=${encodeURIComponent(client.name)}`);
    };

    const onViewContractsAffiliate = () => {
        if (selectedClient) {
            navigate(
                `/contracts?clientName=${encodeURIComponent(selectedClient.name)}`,
            );
        }
    };

    // Server handles filtering, sorting, and pagination — just use the data directly
    const filteredClients = clients;

    if (loading) {
        return (
            <div className="clients-loading">
                <div className="clients-loading-text">
                    Carregando {config.labels.clients.toLowerCase()}...
                </div>
            </div>
        );
    }

    return (
        <div className="clients-container">
            <div className="clients-header">
                <h1 className="clients-title">
                    👥 {config.labels.clients || "Clientes"} e{" "}
                    {config.labels.affiliates || "Afiliados"}
                </h1>
                <div className="button-group">
                    <PrimaryButton onClick={openCreateModal}>
                        + {g.new} {config.labels.client}
                    </PrimaryButton>
                </div>
            </div>

            {error && !showModal && (
                <div className="clients-error">{error}</div>
            )}

            <div className="clients-filters" ref={filtersContainerRef}>
                <button
                    onClick={() => setFilter("all")}
                    className={`clients-filter-button ${filter === "all" ? "active-all" : ""}`}
                >
                    {g.all} ({counts.total})
                </button>
                <button
                    onClick={() => setFilter("active")}
                    className={`clients-filter-button ${filter === "active" ? "active-active" : ""}`}
                >
                    {g.active} ({counts.active})
                </button>
                <button
                    onClick={() => setFilter("inactive")}
                    className={`clients-filter-button ${filter === "inactive" ? "active-inactive" : ""}`}
                >
                    {g.inactive} ({counts.inactive})
                </button>
                <button
                    onClick={() => setFilter("archived")}
                    className={`clients-filter-button ${filter === "archived" ? "active-archived" : ""}`}
                >
                    {g.archived} ({counts.archived})
                </button>

                <input
                    type="text"
                    placeholder="Buscar por nome, apelido, CPF/CNPJ, email..."
                    value={searchTerm}
                    onChange={(e) => setSearchTerm(e.target.value)}
                    className="clients-search-input"
                />
            </div>

            <p className="clients-hint">
                💡 Para visualizar e editar{" "}
                {config.labels.affiliates?.toLowerCase() || "afiliados"}, clique
                em cima do {config.labels.client?.toLowerCase() || "cliente"}{" "}
                desejado na tabela.
            </p>

            <div
                className="clients-table-wrapper"
                style={{
                    opacity: tableLoading ? 0.6 : 1,
                    transition: "opacity 0.15s",
                }}
            >
                <ClientsTable
                    filteredClients={filteredClients}
                    openEditModal={openEditModal}
                    openAffiliatesPanel={openAffiliatesPanel}
                    archiveClient={archiveClient}
                    unarchiveClient={unarchiveClient}
                    onViewContracts={onViewContracts}
                />
            </div>

            <Pagination
                currentPage={currentPage}
                totalItems={totalItems}
                itemsPerPage={itemsPerPage}
                onPageChange={setCurrentPage}
                onItemsPerPageChange={setItemsPerPage}
            />

            <ClientModal
                showModal={showModal}
                modalMode={modalMode}
                formData={formData}
                setFormData={setFormData}
                handleSubmit={handleSubmit}
                closeModal={closeModal}
                error={modalError}
            />

            {showAffiliatesPanel && (
                <AffiliatesPanel
                    selectedClient={selectedClient}
                    affiliates={affiliates}
                    onCreateAffiliate={openCreateAffiliateModal}
                    onEditAffiliate={openEditAffiliateModal}
                    onDeleteAffiliate={deleteAffiliate}
                    onClose={closeAffiliatesPanel}
                    onViewContracts={onViewContractsAffiliate}
                />
            )}

            <AffiliateModal
                showModal={showAffiliateModal}
                modalMode={affiliateModalMode}
                affiliateForm={affiliateForm}
                setAffiliateForm={setAffiliateForm}
                onSubmit={handleAffiliateSubmit}
                onClose={closeAffiliateModal}
                error={affiliateModalError}
            />
        </div>
    );
}
