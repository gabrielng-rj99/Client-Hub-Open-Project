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

import React, { useState, useEffect } from "react";
import { useSearchParams } from "react-router-dom";
import { auditApi } from "../api/auditApi";

import AuditFilters from "../components/audit/AuditFilters";
import AuditLogsTable from "../components/audit/AuditLogsTable";
import Pagination from "../components/common/Pagination";
import PrimaryButton from "../components/common/PrimaryButton";
import "./styles/AuditLogs.css";

export default function AuditLogs({ token, apiUrl, user, onTokenExpired }) {
    const [searchParams, setSearchParams] = useSearchParams();
    const [logs, setLogs] = useState([]);

    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");
    const [totalLogs, setTotalLogs] = useState(0);

    // Derived values from URL
    const currentPage = parseInt(searchParams.get("page") || "1", 10);
    const logsPerPage = parseInt(searchParams.get("limit") || "20", 10);

    // Helper to extract filters from URL
    const getFiltersFromUrl = () => ({
        resource: searchParams.get("resource") || "",
        operation: searchParams.get("operation") || "",
        adminId: searchParams.get("adminId") || "",
        adminSearch: searchParams.get("adminSearch") || "",
        resourceSearch: searchParams.get("resourceSearch") || "",
        changedData: searchParams.get("changedData") || "",
        status: searchParams.get("status") || "",
        ipAddress: searchParams.get("ipAddress") || "",
        resourceId: searchParams.get("resourceId") || "",
        startDate: searchParams.get("startDate") || "",
        endDate: searchParams.get("endDate") || "",
        requestMethod: searchParams.get("requestMethod") || "",
        requestPath: searchParams.get("requestPath") || "",
        responseCode: searchParams.get("responseCode") || "",
        executionTimeMs: searchParams.get("executionTimeMs") || "",
        errorMessage: searchParams.get("errorMessage") || "",
    });

    // Local state for the filter inputs
    const [filters, setFilters] = useState(getFiltersFromUrl());

    // Sync local state when URL changes AND load logs in a single effect
    useEffect(() => {
        setFilters(getFiltersFromUrl());
        loadLogs();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [searchParams]);

    // Related entities are loaded on-demand; no preload on mount.

    const loadLogs = async () => {
        setLoading(true);
        setError("");
        try {
            const offset = (currentPage - 1) * logsPerPage;

            // Use URL params for cleaning/loading, NOT local state
            const currentUrlFilters = getFiltersFromUrl();

            const filterParams = {
                resource: currentUrlFilters.resource || undefined,
                operation: currentUrlFilters.operation || undefined,
                admin_id: currentUrlFilters.adminId || undefined,
                admin_search: currentUrlFilters.adminSearch || undefined,
                resource_search: currentUrlFilters.resourceSearch || undefined,
                changed_data: currentUrlFilters.changedData || undefined,
                status: currentUrlFilters.status || undefined,
                ip_address: currentUrlFilters.ipAddress || undefined,
                resource_id: currentUrlFilters.resourceId || undefined,
                start_date: currentUrlFilters.startDate
                    ? new Date(currentUrlFilters.startDate).toISOString()
                    : undefined,
                end_date: currentUrlFilters.endDate
                    ? new Date(currentUrlFilters.endDate).toISOString()
                    : undefined,
                request_method: currentUrlFilters.requestMethod || undefined,
                request_path: currentUrlFilters.requestPath || undefined,
                response_code: currentUrlFilters.responseCode || undefined,
                execution_time_ms:
                    currentUrlFilters.executionTimeMs || undefined,
                error_message: currentUrlFilters.errorMessage || undefined,
                limit: logsPerPage,
                offset: offset,
            };

            const response = await auditApi.getAuditLogs(
                apiUrl,
                token,
                filterParams,
                onTokenExpired,
            );
            setLogs(response.data || []);
            setTotalLogs(response.total || 0);
        } catch (err) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    };

    const handleApplyFilters = (filtersToApply = null) => {
        // Use passed filters if provided, otherwise use current state
        const filtersObj = filtersToApply || filters;

        setSearchParams((prev) => {
            const newParams = new URLSearchParams(prev);

            // Update all filter keys
            Object.entries(filtersObj).forEach(([key, value]) => {
                if (value) {
                    newParams.set(key, value);
                } else {
                    newParams.delete(key);
                }
            });

            // Reset to page 1 on filter apply
            newParams.set("page", "1");
            return newParams;
        });
    };

    const setCurrentPage = (page) => {
        setSearchParams((prev) => {
            const newParams = new URLSearchParams(prev);
            newParams.set("page", page.toString());
            return newParams;
        });
    };

    const setLogsPerPage = (limit) => {
        setSearchParams((prev) => {
            const newParams = new URLSearchParams(prev);
            newParams.set("limit", limit.toString());
            newParams.set("page", "1");
            return newParams;
        });
    };

    const handleExport = async () => {
        try {
            // Use current URL filters for export too
            const currentUrlFilters = getFiltersFromUrl();
            const filterParams = {
                resource: currentUrlFilters.resource || undefined,
                operation: currentUrlFilters.operation || undefined,
                admin_id: currentUrlFilters.adminId || undefined,
                admin_search: currentUrlFilters.adminSearch || undefined,
                resource_search: currentUrlFilters.resourceSearch || undefined,
                changed_data: currentUrlFilters.changedData || undefined,
                request_method: currentUrlFilters.requestMethod || undefined,
                request_path: currentUrlFilters.requestPath || undefined,
                response_code: currentUrlFilters.responseCode || undefined,
                execution_time_ms:
                    currentUrlFilters.executionTimeMs || undefined,
                error_message: currentUrlFilters.errorMessage || undefined,
            };

            const data = await auditApi.exportAuditLogs(
                apiUrl,
                token,
                filterParams,
                onTokenExpired,
            );
            const json = JSON.stringify(data, null, 2);
            const blob = new Blob([json], { type: "application/json" });
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement("a");
            a.href = url;
            a.download = `audit_logs_${new Date().getTime()}.json`;
            a.click();
            window.URL.revokeObjectURL(url);
        } catch (err) {
            setError("Erro ao exportar logs");
        }
    };

    if (!user || !user?.resources?.['audit_logs']?.includes('read')) {
        return (
            <div className="audit-logs-access-denied">
                <div className="audit-logs-access-denied-text">
                    Acesso negado. Você não tem permissão para acessar logs de auditoria.
                </div>
            </div>
        );
    }

    return (
        <div className="audit-logs-container">
            <div className="audit-logs-header">
                <h1 className="audit-logs-title">🔍 Logs</h1>
                <div className="button-group">
                    <PrimaryButton
                        onClick={handleExport}
                        style={{ minWidth: "160px" }}
                        disabled={!user?.resources?.['audit_logs']?.includes('export')}
                    >
                        📥 Exportar JSON
                    </PrimaryButton>
                </div>
            </div>

            {error && <div className="audit-logs-error">{error}</div>}

            <AuditFilters
                filters={filters}
                setFilters={setFilters}
                onApply={handleApplyFilters}
            />

            <div className="audit-logs-table-wrapper">
                <AuditLogsTable logs={logs} loading={loading} />
            </div>

            <Pagination
                currentPage={currentPage}
                totalItems={totalLogs}
                itemsPerPage={logsPerPage}
                onPageChange={setCurrentPage}
                onItemsPerPageChange={setLogsPerPage}
            />
        </div>
    );
}
