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

import React, { useState, useMemo, useEffect, useRef } from "react";
import Select from "react-select";
import AsyncSelect from "react-select/async";
import { useConfig } from "../../contexts/ConfigContext";
import FinancialForm from "../financial/FinancialForm";
import "./ContractsModal.css";

/**
 * ContractModal — create/edit modal.
 *
 * Performance optimisations (Issue #3):
 *  - "Cliente" uses AsyncSelect with server-side search (limit=20).
 *    The full clients list is NOT loaded; we only fetch what matches the query.
 *  - "Categoria" still uses the pre-loaded list (usually small, ≤100 entries).
 *  - "Subcategoria" is derived from the selected category (filtered subset).
 *
 * Pre-fill fix (Issue #2):
 *  - category_id is auto-derived from categories × subcategory_id so the
 *    correct category is selected when opening the edit modal even when
 *    the contract object does not have a `line` property.
 *
 * Dates (Issue #1 related):
 *  - Native <input type="date"> instead of manual text formatting.
 */
export default function ContractModal({
    showModal,
    modalMode,
    formData,
    setFormData,
    clients,        // Used ONLY as fallback for the initial selected-value label
    categories,
    lines,
    affiliates,
    onSubmit,
    onClose,
    onCategoryChange,
    onClientChange,
    error,
    // Search callback — parent provides (query) => Promise<[{id, name, nickname}]>
    onSearchClients,
    // Financial props
    financialData = null,
    onFinancialChange = null,
    showFinancialSection = true,
    showFinancialValues = true,
    canEditFinancialValues = true,
    onSearchCategories,
}) {
    const { config } = useConfig();
    const { labels } = config;

    if (!showModal) return null;

    // ── react-select shared styles ──────────────────────────────────────────
    const customSelectStyles = {
        control: (provided, state) => ({
            ...provided,
            width: "100%",
            fontSize: "14px",
            border: "1px solid #ced4da",
            borderRadius: "4px",
            background: "white",
            color: "#333",
            cursor: "pointer",
            boxShadow: state.isFocused ? "0 0 0 1px #3498db" : provided.boxShadow,
            "&:hover": { borderColor: "#3498db" },
        }),
        option: (provided, state) => ({
            ...provided,
            backgroundColor: state.isSelected
                ? "#3498db"
                : state.isFocused
                    ? "#f8f9fa"
                    : "white",
            color: state.isSelected ? "white" : "#333",
            cursor: "pointer",
        }),
        menu: (provided) => ({
            ...provided,
            background: "white",
            border: "1px solid #ced4da",
            borderRadius: "4px",
            zIndex: 9999,
        }),
        singleValue: (provided) => ({ ...provided, color: "#333" }),
        placeholder: (provided) => ({ ...provided, color: "#999" }),
        input: (provided) => ({ ...provided, color: "#333" }),
    };

    const labelStyle = {
        display: "block",
        marginBottom: "8px",
        fontSize: "14px",
        fontWeight: "500",
        color: "#495057",
    };

    const optionalSpan = (
        <span style={{ fontSize: "12px", color: "#7f8c8d", marginLeft: "4px" }}>
            (opcional)
        </span>
    );

    // ── Labels ──────────────────────────────────────────────────────────────
    const clientLabel = labels.client || "Cliente";
    const affiliateLabel = labels.affiliate || "Afiliado";
    const categoryLabel = labels.category || "Categoria";
    const subcategoryLabel = labels.subcategory || "Subcategoria";
    const modelLabel = labels.model || "Descrição";
    const itemKeyLabel = labels.item_key || "Identificador";
    const contractLabel = labels.contract || "Contrato";

    // ── Derive category_id when editing (fix for Issue #2) ──────────────────
    // When categories are loaded and the formData has a subcategory_id but no
    // category_id, we look up the parent category automatically. This allows
    // the edit modal to pre-fill the category select correctly even when the
    // contract object comes from GetAllContractsIncludingArchived (which doesn't
    // embed a full `line` object with category_id).
    const derivedCategoryId = useMemo(() => {
        if (formData.category_id) return formData.category_id;
        if (!formData.subcategory_id) return "";
        for (const cat of categories) {
            if (Array.isArray(cat.lines)) {
                if (cat.lines.some((l) => l.id === formData.subcategory_id)) {
                    return cat.id;
                }
            }
        }
        return "";
    }, [formData.category_id, formData.subcategory_id, categories]);

    // Ensure the form always has the derived category_id so Subcategory select
    // can find its options. We do this via a stable variable rather than a
    // useState to avoid triggering extra renders.
    const effectiveCategoryId = derivedCategoryId;

    // ── Pagination State for Clients ─────────────────────────────────────────
    const [clientOpts, setClientOptions] = useState([]);
    const [clientLoading, setClientLoading] = useState(false);
    const [clientSearch, setClientSearch] = useState("");
    const [clientOffset, setClientOffset] = useState(0);
    const [clientHasMore, setClientHasMore] = useState(true);
    const clientSearchTimeout = useRef(null);

    const loadClientPage = async (query = "", offset = 0, isNewSearch = false) => {
        setClientLoading(true);
        try {
            let formatted = [];
            if (!onSearchClients) {
                const q = query.toLowerCase();
                formatted = clients
                    .filter(c => !c.archived_at)
                    .filter(c => c.name.toLowerCase().includes(q) || (c.nickname || "").toLowerCase().includes(q))
                    .slice(0, 100)
                    .map(c => ({ value: c.id, label: `${c.name}${c.nickname ? ` (${c.nickname})` : ""}` }));
            } else {
                const results = await onSearchClients(query, offset);
                formatted = results.map(c => ({ value: c.id, label: `${c.name}${c.nickname ? ` (${c.nickname})` : ""}` }));
            }

            if (isNewSearch) {
                setClientOptions(formatted);
            } else {
                setClientOptions(prev => {
                    const existingIds = new Set(prev.map(p => p.value));
                    const newItems = formatted.filter(f => !existingIds.has(f.value));
                    return [...prev, ...newItems];
                });
            }
            setClientHasMore(formatted.length === 100);
        } catch (e) {
            console.error(e);
        } finally {
            setClientLoading(false);
        }
    };

    useEffect(() => {
        if (showModal) {
            setClientSearch("");
            setClientOffset(0);
            loadClientPage("", 0, true);
        }
    }, [showModal]);

    const handleClientScrollToBottom = () => {
        if (!clientLoading && clientHasMore) {
            const nextOffset = clientOffset + 100;
            setClientOffset(nextOffset);
            loadClientPage(clientSearch, nextOffset, false);
        }
    };

    const handleClientInputChange = (val, actionMeta) => {
        if (actionMeta.action === "input-change") {
            setClientSearch(val);
            if (clientSearchTimeout.current) clearTimeout(clientSearchTimeout.current);
            clientSearchTimeout.current = setTimeout(() => {
                setClientOffset(0);
                loadClientPage(val, 0, true);
            }, 300);
        }
    };

    // ── Pagination State for Categories ──────────────────────────────────────
    const [catOptions, setCatOptions] = useState([]);
    const [catLoading, setCatLoading] = useState(false);
    const [catSearch, setCatSearch] = useState("");
    const [catOffset, setCatOffset] = useState(0);
    const [catHasMore, setCatHasMore] = useState(true);
    const catSearchTimeout = useRef(null);

    const loadCategoryPage = async (query = "", offset = 0, isNewSearch = false) => {
        setCatLoading(true);
        try {
            let formatted = [];
            if (!onSearchCategories) {
                const q = query.toLowerCase();
                formatted = categories
                    .filter(cat => !cat.archived_at)
                    .filter(cat => cat.name.toLowerCase().includes(q))
                    .slice(0, 100)
                    .map(cat => ({ value: cat.id, label: cat.name }));
            } else {
                const results = await onSearchCategories(query, offset);
                formatted = results.map(c => ({ value: c.id, label: c.name }));
            }

            if (isNewSearch) {
                setCatOptions(formatted);
            } else {
                setCatOptions(prev => {
                    const existingIds = new Set(prev.map(p => p.value));
                    const newItems = formatted.filter(f => !existingIds.has(f.value));
                    return [...prev, ...newItems];
                });
            }
            setCatHasMore(formatted.length === 100);
        } catch (e) {
            console.error(e);
        } finally {
            setCatLoading(false);
        }
    };

    useEffect(() => {
        if (showModal) {
            setCatSearch("");
            setCatOffset(0);
            loadCategoryPage("", 0, true);
        }
    }, [showModal]);

    const handleCatScrollToBottom = () => {
        if (!catLoading && catHasMore) {
            const nextOffset = catOffset + 100;
            setCatOffset(nextOffset);
            loadCategoryPage(catSearch, nextOffset, false);
        }
    };

    const handleCatInputChange = (val, actionMeta) => {
        if (actionMeta.action === "input-change") {
            setCatSearch(val);
            if (catSearchTimeout.current) clearTimeout(catSearchTimeout.current);
            catSearchTimeout.current = setTimeout(() => {
                setCatOffset(0);
                loadCategoryPage(val, 0, true);
            }, 300);
        }
    };

    // Build the currently-selected client option for the AsyncSelect
    const selectedClientOption = useMemo(() => {
        if (!formData.client_id) return null;
        // Try the pre-loaded clients list first (fast)
        const found = clients.find((c) => c.id === formData.client_id);
        if (found) {
            return {
                value: found.id,
                label: `${found.name}${found.nickname ? ` (${found.nickname})` : ""}`,
            };
        }
        // Fallback: show the ID (will be replaced once user opens the dropdown)
        return { value: formData.client_id, label: formData.client_id };
    }, [formData.client_id, clients]);

    // Build the currently-selected category option
    const selectedCategoryOption = useMemo(() => {
        if (!effectiveCategoryId) return null;
        const found = categories.find((c) => c.id === effectiveCategoryId);
        if (found) {
            return {
                value: found.id,
                label: found.name,
            };
        }
        return { value: effectiveCategoryId, label: effectiveCategoryId };
    }, [effectiveCategoryId, categories]);

    // ── Subcategory options (filtered by category) ───────────────────────────
    const subcategoryOptions = useMemo(
        () =>
            lines.map((l) => ({ value: l.id, label: l.name })),
        [lines],
    );

    const selectedSubcategoryOption = useMemo(() => {
        if (!formData.subcategory_id) return null;
        return (
            subcategoryOptions.find((o) => o.value === formData.subcategory_id) ||
            null
        );
    }, [formData.subcategory_id, subcategoryOptions]);

    // ── Date helpers: store as yyyy-mm-dd (native date input format) ─────────
    // formData.start_date / end_date can arrive as "dd/mm/yyyy" (legacy) or
    // "yyyy-mm-dd" (native). We normalise to yyyy-mm-dd for the input value.
    const toInputDate = (val) => {
        if (!val) return "";
        // Already yyyy-mm-dd
        if (/^\d{4}-\d{2}-\d{2}$/.test(val)) return val;
        // dd/mm/yyyy → yyyy-mm-dd
        const parts = val.split("/");
        if (parts.length === 3) {
            const [d, m, y] = parts;
            if (y && y.length === 4)
                return `${y}-${m.padStart(2, "0")}-${d.padStart(2, "0")}`;
        }
        return val;
    };

    const inputStyle = {
        width: "100%",
        padding: "10px",
        border: "1px solid #ced4da",
        borderRadius: "4px",
        fontSize: "14px",
        boxSizing: "border-box",
    };

    return (
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
                overflowY: "auto",
                padding: "20px",
            }}
            onClick={onClose}
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
                    {modalMode === "create"
                        ? `Novo ${contractLabel}`
                        : `Editar ${contractLabel}`}
                </h2>

                {error && (
                    <div
                        style={{
                            background: "#fee",
                            color: "#c33",
                            padding: "12px 16px",
                            borderRadius: "4px",
                            border: "1px solid #fcc",
                            marginBottom: "20px",
                            fontSize: "14px",
                        }}
                    >
                        {error}
                    </div>
                )}

                <form onSubmit={onSubmit}>
                    <div style={{ display: "grid", gap: "20px" }}>

                        {/* ── Cliente (AsyncSelect) ────────────────────────── */}
                        <div>
                            <label style={labelStyle}>{clientLabel} *</label>
                            <Select
                                options={clientOpts}
                                onInputChange={handleClientInputChange}
                                onMenuScrollToBottom={handleClientScrollToBottom}
                                value={selectedClientOption}
                                onChange={(selected) => {
                                    setFormData({
                                        ...formData,
                                        client_id: selected ? selected.value : "",
                                        affiliate_id: "",
                                    });
                                    if (onClientChange) onClientChange(selected ? selected.value : "");
                                }}
                                isClearable
                                isSearchable
                                filterOption={null}
                                placeholder={`Buscar ${clientLabel.toLowerCase()}…`}
                                noOptionsMessage={() => "Nenhum cliente encontrado"}
                                loadingMessage={() => "Buscando…"}
                                styles={customSelectStyles}
                            />
                        </div>

                        {/* ── Afiliado ─────────────────────────────────────── */}
                        <div>
                            <label style={labelStyle}>
                                {affiliateLabel}
                                {optionalSpan}
                            </label>
                            <Select
                                value={
                                    formData.affiliate_id
                                        ? {
                                            value: formData.affiliate_id,
                                            label:
                                                affiliates.find(
                                                    (a) => a.id === formData.affiliate_id,
                                                )?.name || formData.affiliate_id,
                                        }
                                        : null
                                }
                                onChange={(selected) =>
                                    setFormData({
                                        ...formData,
                                        affiliate_id: selected ? selected.value : "",
                                    })
                                }
                                options={[
                                    {
                                        value: "",
                                        label: `Nenhum ${affiliateLabel.toLowerCase()}`,
                                    },
                                    ...affiliates.map((aff) => ({
                                        value: aff.id,
                                        label: aff.name,
                                    })),
                                ]}
                                isSearchable
                                isDisabled={!formData.client_id || affiliates.length === 0}
                                placeholder={`Selecione um ${affiliateLabel.toLowerCase()}`}
                                styles={{
                                    ...customSelectStyles,
                                    control: (p, s) => ({
                                        ...customSelectStyles.control(p, s),
                                        opacity:
                                            !formData.client_id || affiliates.length === 0
                                                ? 0.6
                                                : 1,
                                    }),
                                }}
                            />
                        </div>

                        {/* ── Categoria ────────────────────────────────────── */}
                        <div>
                            <label style={labelStyle}>{categoryLabel} *</label>
                            <Select
                                options={catOptions}
                                onInputChange={handleCatInputChange}
                                onMenuScrollToBottom={handleCatScrollToBottom}
                                value={selectedCategoryOption}
                                onChange={(selected) => {
                                    setFormData({
                                        ...formData,
                                        category_id: selected ? selected.value : "",
                                        subcategory_id: "",
                                    });
                                    if (onCategoryChange) onCategoryChange(selected ? selected.value : "");
                                }}
                                isClearable
                                isSearchable
                                filterOption={null}
                                placeholder={`Selecione uma ${categoryLabel.toLowerCase()}...`}
                                loadingMessage={() => "Carregando..."}
                                noOptionsMessage={() => "Nenhuma categoria encontrada"}
                                styles={customSelectStyles}
                            />
                        </div>

                        {/* ── Subcategoria ─────────────────────────────────── */}
                        <div>
                            <label style={labelStyle}>{subcategoryLabel} *</label>
                            <Select
                                value={selectedSubcategoryOption}
                                onChange={(selected) =>
                                    setFormData({
                                        ...formData,
                                        subcategory_id: selected ? selected.value : "",
                                    })
                                }
                                options={[
                                    {
                                        value: "",
                                        label: `Selecione uma ${subcategoryLabel.toLowerCase()}`,
                                    },
                                    ...subcategoryOptions,
                                ]}
                                isSearchable
                                isDisabled={!effectiveCategoryId || lines.length === 0}
                                placeholder={`Selecione uma ${subcategoryLabel.toLowerCase()}`}
                                styles={{
                                    ...customSelectStyles,
                                    control: (p, s) => ({
                                        ...customSelectStyles.control(p, s),
                                        opacity:
                                            !effectiveCategoryId || lines.length === 0
                                                ? 0.6
                                                : 1,
                                    }),
                                }}
                                required
                            />
                        </div>

                        {/* ── Modelo/Descrição ─────────────────────────────── */}
                        <div>
                            <label style={labelStyle}>
                                {modelLabel}
                                {optionalSpan}
                            </label>
                            <input
                                type="text"
                                value={formData.model}
                                onChange={(e) =>
                                    setFormData({ ...formData, model: e.target.value })
                                }
                                placeholder="Ex: Plano Básico, Premium, etc."
                                style={inputStyle}
                            />
                        </div>

                        {/* ── Identificador ────────────────────────────────── */}
                        <div>
                            <label style={labelStyle}>
                                {itemKeyLabel}
                                {optionalSpan}
                            </label>
                            <input
                                type="text"
                                value={formData.item_key}
                                onChange={(e) =>
                                    setFormData({ ...formData, item_key: e.target.value })
                                }
                                placeholder="Ex: KEY-12345"
                                style={inputStyle}
                            />
                        </div>

                        {/* ── Datas (native date inputs) ────────────────────── */}
                        <div
                            style={{
                                display: "grid",
                                gridTemplateColumns: "1fr 1fr",
                                gap: "16px",
                            }}
                        >
                            <div>
                                <label style={labelStyle}>
                                    Data de Início{optionalSpan}
                                </label>
                                <input
                                    type="date"
                                    value={toInputDate(formData.start_date)}
                                    onChange={(e) =>
                                        setFormData({
                                            ...formData,
                                            start_date: e.target.value,
                                        })
                                    }
                                    style={inputStyle}
                                />
                            </div>
                            <div>
                                <label style={labelStyle}>
                                    Data de Vencimento{optionalSpan}
                                </label>
                                <input
                                    type="date"
                                    value={toInputDate(formData.end_date)}
                                    onChange={(e) =>
                                        setFormData({
                                            ...formData,
                                            end_date: e.target.value,
                                        })
                                    }
                                    style={inputStyle}
                                />
                            </div>
                        </div>

                        <div style={{ fontSize: "12px", color: "#7f8c8d", marginTop: "-10px" }}>
                            <p style={{ margin: 0 }}>
                                💡 Dica: As datas são opcionais. {contractLabel}s sem data de
                                término são considerados permanentes (ex: licenças vitalícias).
                            </p>
                        </div>

                        {/* ── Financial Section ─────────────────────────────── */}
                        {showFinancialSection && onFinancialChange && (
                            <FinancialForm
                                financialData={financialData}
                                onChange={onFinancialChange}
                                disabled={false}
                                showValues={showFinancialValues}
                                canEditValues={canEditFinancialValues}
                            />
                        )}
                    </div>

                    <div
                        style={{
                            display: "flex",
                            gap: "12px",
                            justifyContent: "flex-end",
                            marginTop: "32px",
                        }}
                    >
                        <button
                            type="button"
                            onClick={onClose}
                            style={{
                                padding: "10px 24px",
                                background: "#95a5a6",
                                color: "white",
                                border: "none",
                                borderRadius: "4px",
                                cursor: "pointer",
                                fontSize: "14px",
                            }}
                        >
                            Cancelar
                        </button>
                        <button
                            type="submit"
                            style={{
                                padding: "10px 24px",
                                background: "#27ae60",
                                color: "white",
                                border: "none",
                                borderRadius: "4px",
                                cursor: "pointer",
                                fontSize: "14px",
                                fontWeight: "600",
                            }}
                        >
                            {modalMode === "create"
                                ? `Criar ${contractLabel}`
                                : "Salvar Alterações"}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
}
