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

import React from "react";
import {
    formatDate,
    getContractStatus,
    getClientName,
} from "../../utils/contractHelpers";
import "./ContractsTable.css";
import ArchiveIcon from "../../assets/icons/archive.svg";
import UnarchiveIcon from "../../assets/icons/unarchive.svg";
import TrashIcon from "../../assets/icons/trash.svg";

import { useConfig } from "../../contexts/ConfigContext";

export default function ContractsTable({
    filteredContracts,
    clients,
    categories,
    onEdit,
    onArchive,
    onUnarchive,
    onDelete,
}) {
    const { config } = useConfig();
    const { labels } = config;

    // Fallback Maps only built if enriched fields are missing (backward compat)
    const categoryBySubcategoryId = React.useMemo(() => {
        const map = new Map();
        for (const category of categories) {
            if (!Array.isArray(category.lines)) continue;
            for (const line of category.lines) {
                map.set(line.id, category.name);
            }
        }
        return map;
    }, [categories]);

    const subcategoryNameById = React.useMemo(() => {
        const map = new Map();
        for (const category of categories) {
            if (!Array.isArray(category.lines)) continue;
            for (const line of category.lines) {
                map.set(line.id, line.name);
            }
        }
        return map;
    }, [categories]);

    // Use enriched fields when available, fallback to Map lookup
    const getCategoryName = (contract) => {
        if (contract.category_name) return contract.category_name;
        if (!contract.subcategory_id) return "-";
        return categoryBySubcategoryId.get(contract.subcategory_id) || "-";
    };

    const getSubcategoryName = (contract) => {
        if (contract.subcategory_name) return contract.subcategory_name;
        if (!contract.subcategory_id) return "-";
        return subcategoryNameById.get(contract.subcategory_id) || "-";
    };

    const getContractClientName = (contract) => {
        if (contract.client_name) return contract.client_name;
        return getClientName(contract.client_id, clients);
    };

    if (filteredContracts.length === 0) {
        return (
            <div className="contracts-table-empty">
                <p>
                    Nenhum {labels.contract?.toLowerCase() || "acordo"}{" "}
                    encontrado
                </p>
            </div>
        );
    }

    return (
        <table className="contracts-table">
            <thead>
                <tr>
                    <th>{labels.model || "Modelo"}</th>
                    <th>{labels.client || "Cliente"}</th>
                    <th>{labels.category || "Categoria"}</th>
                    <th>{labels.subcategory || "Subcategoria"}</th>
                    <th>Início</th>
                    <th>Vencimento</th>
                    <th className="status">Status</th>
                    <th className="actions">Ações</th>
                </tr>
            </thead>
            <tbody>
                {filteredContracts.map((contract) => {
                    const status = getContractStatus(contract);
                    const isArchived = !!contract.archived_at;

                    // Status badge: archived > computed
                    const statusLabel = isArchived
                        ? "Arquivado"
                        : status.status;
                    const statusColor = isArchived
                        ? "#95a5a6"
                        : status.color;

                    return (
                        <tr
                            key={contract.id}
                            className={`${isArchived ? "archived" : ""} contracts-table-row-clickable`}
                            onClick={() => onEdit(contract)}
                            title="Clique para abrir detalhes"
                        >
                            <td>
                                <div className="contracts-table-model">
                                    {contract.model || "-"}
                                </div>
                                {contract.item_key && (
                                    <div className="contracts-table-product-key">
                                        {contract.item_key}
                                    </div>
                                )}
                            </td>
                            <td>
                                {getContractClientName(contract)}
                                {contract.affiliate && (
                                    <div className="contracts-table-affiliate">
                                        {labels.affiliate || "Afiliado"}:{" "}
                                        {contract.affiliate.name}
                                    </div>
                                )}
                            </td>
                            <td>{getCategoryName(contract)}</td>
                            <td>{getSubcategoryName(contract)}</td>
                            <td>{formatDate(contract.start_date)}</td>
                            <td>{formatDate(contract.end_date)}</td>
                            <td className="status">
                                <span
                                    className="contracts-table-status"
                                    style={{
                                        background: `${statusColor}20`,
                                        color: statusColor,
                                    }}
                                >
                                    {statusLabel}
                                </span>
                            </td>
                            <td
                                className="actions"
                                onClick={(e) => e.stopPropagation()}
                            >
                                <div className="contracts-table-actions">
                                    {/* Delete */}
                                    <button
                                        onClick={() => onDelete?.(contract)}
                                        className="contracts-table-icon-button"
                                        title="Excluir contrato"
                                    >
                                        <img
                                            src={TrashIcon}
                                            alt="Deletar"
                                            className="contracts-table-icon delete-icon"
                                        />
                                    </button>

                                    {/* Archive / Unarchive */}
                                    {!isArchived && (
                                        <button
                                            onClick={() =>
                                                onArchive(contract.id)
                                            }
                                            className="contracts-table-icon-button"
                                            title="Arquivar"
                                        >
                                            <img
                                                src={ArchiveIcon}
                                                alt="Arquivar"
                                                style={{
                                                    width: "22px",
                                                    height: "22px",
                                                    filter: "invert(64%) sepia(81%) saturate(455%) hue-rotate(359deg) brightness(98%) contrast(91%)",
                                                }}
                                            />
                                        </button>
                                    )}
                                    {isArchived && (
                                        <button
                                            onClick={() =>
                                                onUnarchive(contract.id)
                                            }
                                            className="contracts-table-icon-button"
                                            title="Desarquivar"
                                        >
                                            <img
                                                src={UnarchiveIcon}
                                                alt="Desarquivar"
                                                style={{
                                                    width: "22px",
                                                    height: "22px",
                                                    filter: "invert(62%) sepia(34%) saturate(760%) hue-rotate(88deg) brightness(93%) contrast(81%)",
                                                }}
                                            />
                                        </button>
                                    )}
                                </div>
                            </td>
                        </tr>
                    );
                })}
            </tbody>
        </table>
    );
}
