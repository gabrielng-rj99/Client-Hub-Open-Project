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
import Select from "react-select";
import "./FinancialForm.css";

/**
 * FinancialForm - Componente para gerenciar modelo de financeiro de contratos
 *
 * Props:
 * - financialData: Dados do financeiro existente (ou null para novo)
 * - onChange: Callback quando os dados mudam
 * - disabled: Se o formulário está desabilitado
 * - showValues: Se deve mostrar os campos de valores (permissão)
 * - canEditValues: Se pode editar os valores (permissão)
 */
export default function FinancialForm({
    financialData,
    onChange,
    disabled = false,
    showValues = true,
    canEditValues = true,
}) {
    const [financialType, setFinancialType] = useState(
        financialData?.financial_type || "unico"
    );
    const [recurrenceType, setRecurrenceType] = useState(
        financialData?.recurrence_type || "mensal"
    );
    const [dueDay, setDueDay] = useState(financialData?.due_day || 10);
    const [clientValue, setClientValue] = useState(
        financialData?.client_value || ""
    );
    const [receivedValue, setReceivedValue] = useState(
        financialData?.received_value || ""
    );
    const [description, setDescription] = useState(
        financialData?.description || ""
    );
    const [installments, setInstallments] = useState(
        financialData?.installments || [
            { installment_number: 0, installment_label: "Entrada", client_value: "", received_value: "" },
        ]
    );

    // Notificar mudanças
    useEffect(() => {
        const data = {
            financial_type: financialType,
            recurrence_type: financialType === "recorrente" ? recurrenceType : null,
            due_day: financialType === "recorrente" ? dueDay : null,
            client_value: financialType !== "personalizado" ? parseFloat(clientValue) || null : null,
            received_value: financialType !== "personalizado" ? parseFloat(receivedValue) || null : null,
            description: description || null,
            installments: financialType === "personalizado" ? installments.map((inst, idx) => ({
                ...inst,
                installment_number: idx,
                client_value: parseFloat(inst.client_value) || 0,
                received_value: parseFloat(inst.received_value) || 0,
            })) : [],
        };
        onChange?.(data);
    }, [financialType, recurrenceType, dueDay, clientValue, receivedValue, description, installments]);

    // Opções de tipo de financeiro
    const financialTypeOptions = [
        { value: "unico", label: "Financeiro Único" },
        { value: "recorrente", label: "Recorrente (Mensalidade)" },
        { value: "personalizado", label: "Personalizado (Parcelas)" },
    ];

    // Opções de recorrência
    const recurrenceOptions = [
        { value: "mensal", label: "Mensal" },
        { value: "trimestral", label: "Trimestral" },
        { value: "semestral", label: "Semestral" },
        { value: "anual", label: "Anual" },
    ];

    // Adicionar parcela
    const addInstallment = () => {
        const nextNumber = installments.length;
        const label = nextNumber === 0 ? "Entrada" : `${nextNumber}ª Parcela`;
        setInstallments([
            ...installments,
            { installment_number: nextNumber, installment_label: label, client_value: "", received_value: "" },
        ]);
    };

    // Remover parcela
    const removeInstallment = (index) => {
        if (installments.length <= 1) return;
        const newInstallments = installments.filter((_, i) => i !== index);
        // Reordenar números
        setInstallments(
            newInstallments.map((inst, idx) => ({
                ...inst,
                installment_number: idx,
                installment_label: idx === 0 ? "Entrada" : `${idx}ª Parcela`,
            }))
        );
    };

    // Atualizar parcela
    const updateInstallment = (index, field, value) => {
        const newInstallments = [...installments];
        newInstallments[index] = { ...newInstallments[index], [field]: value };
        setInstallments(newInstallments);
    };

    // Calcular totais das parcelas
    const totalClientValue = installments.reduce(
        (sum, inst) => sum + (parseFloat(inst.client_value) || 0),
        0
    );
    const totalReceivedValue = installments.reduce(
        (sum, inst) => sum + (parseFloat(inst.received_value) || 0),
        0
    );

    // Formatar valor monetário
    const formatCurrency = (value) => {
        if (!value && value !== 0) return "-";
        return new Intl.NumberFormat("pt-BR", {
            style: "currency",
            currency: "BRL",
        }).format(value);
    };

    // Estilos para react-select
    const customSelectStyles = {
        control: (provided, state) => ({
            ...provided,
            width: "100%",
            fontSize: "14px",
            border: "1px solid var(--border-color, #ddd)",
            borderRadius: "4px",
            background: disabled ? "var(--disabled-bg, #f5f5f5)" : "var(--content-bg, white)",
            color: "var(--primary-text-color, #333)",
            cursor: disabled ? "not-allowed" : "pointer",
            opacity: disabled ? 0.7 : 1,
        }),
        option: (provided, state) => ({
            ...provided,
            backgroundColor: state.isSelected
                ? "var(--primary-color, #3498db)"
                : state.isFocused
                    ? "var(--hover-bg, #f8f9fa)"
                    : "var(--content-bg, white)",
            color: state.isSelected ? "white" : "var(--primary-text-color, #333)",
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

    return (
        <div className="financial-form">
            <div className="financial-form-section">
                <h4 className="financial-form-section-title">
                    💰 Modelo de Financeiro
                    <span className="financial-form-optional">(opcional)</span>
                </h4>

                {/* Tipo de Financeiro */}
                <div className="financial-form-field">
                    <label>Tipo de Financeiro</label>
                    <Select
                        value={financialTypeOptions.find((opt) => opt.value === financialType)}
                        onChange={(opt) => setFinancialType(opt.value)}
                        options={financialTypeOptions}
                        styles={customSelectStyles}
                        isDisabled={disabled}
                        placeholder="Selecione..."
                    />
                </div>

                {/* Descrição contextual do tipo de financeiro */}
                <div className="financial-form-type-hint">
                    {financialType === "unico" && (
                        <span>💡 O lançamento aparecerá no painel financeiro na data de vencimento definida abaixo.</span>
                    )}
                    {financialType === "recorrente" && recurrenceType === "mensal" && (
                        <span>💡 Parcelas geradas no dia {dueDay} de cada mês, a partir do início do contrato.</span>
                    )}
                    {financialType === "recorrente" && recurrenceType === "trimestral" && (
                        <span>💡 Parcelas geradas a cada 3 meses no dia {dueDay}, a partir do início do contrato.</span>
                    )}
                    {financialType === "recorrente" && recurrenceType === "semestral" && (
                        <span>💡 Parcelas geradas a cada 6 meses no dia {dueDay}, a partir do início do contrato.</span>
                    )}
                    {financialType === "recorrente" && recurrenceType === "anual" && (
                        <span>💡 Parcela anual gerada no dia {dueDay} do mês de início do contrato.</span>
                    )}
                    {financialType === "personalizado" && (
                        <span>💡 Cada parcela aparecerá no painel financeiro na data específica que você definir em cada item.</span>
                    )}
                </div>

                {/* Campos para Recorrente */}
                {financialType === "recorrente" && (
                    <div className="financial-form-row">
                        <div className="financial-form-field">
                            <label>Periodicidade</label>
                            <Select
                                value={recurrenceOptions.find((opt) => opt.value === recurrenceType)}
                                onChange={(opt) => setRecurrenceType(opt.value)}
                                options={recurrenceOptions}
                                styles={customSelectStyles}
                                isDisabled={disabled}
                            />
                        </div>
                        <div className="financial-form-field">
                            <label>Dia do Vencimento</label>
                            <input
                                type="number"
                                min="1"
                                max="31"
                                value={dueDay}
                                onChange={(e) => setDueDay(parseInt(e.target.value) || 10)}
                                disabled={disabled}
                                className="financial-form-input"
                            />
                        </div>
                    </div>
                )}

                {/* Campos de Valor (para Único e Recorrente) */}
                {financialType !== "personalizado" && showValues && (
                    <div className="financial-form-row">
                        <div className="financial-form-field">
                            <label>Valor Cliente Paga (R$)</label>
                            <input
                                type="number"
                                step="0.01"
                                min="0"
                                value={clientValue}
                                onChange={(e) => setClientValue(e.target.value)}
                                disabled={disabled || !canEditValues}
                                placeholder="0,00"
                                className="financial-form-input"
                            />
                        </div>
                        <div className="financial-form-field">
                            <label>Valor Você Recebe (R$)</label>
                            <input
                                type="number"
                                step="0.01"
                                min="0"
                                value={receivedValue}
                                onChange={(e) => setReceivedValue(e.target.value)}
                                disabled={disabled || !canEditValues}
                                placeholder="0,00"
                                className="financial-form-input"
                            />
                        </div>
                    </div>
                )}

                {/* Parcelas Personalizadas */}
                {financialType === "personalizado" && (
                    <div className="financial-form-installments">
                        <div className="financial-form-installments-header">
                            <span>Parcelas</span>
                            <button
                                type="button"
                                onClick={addInstallment}
                                disabled={disabled}
                                className="financial-form-add-btn"
                            >
                                + Adicionar Parcela
                            </button>
                        </div>

                        <div className="financial-form-installments-list">
                            {installments.map((inst, index) => (
                                <div key={index} className="financial-form-installment-row">
                                    <div className="financial-form-installment-label">
                                        <input
                                            type="text"
                                            value={inst.installment_label || ""}
                                            onChange={(e) =>
                                                updateInstallment(index, "installment_label", e.target.value)
                                            }
                                            disabled={disabled}
                                            placeholder={index === 0 ? "Entrada" : `${index}ª Parcela`}
                                            className="financial-form-input financial-form-input-label"
                                        />
                                    </div>
                                    {showValues && (
                                        <>
                                            <div className="financial-form-installment-value">
                                                <input
                                                    type="number"
                                                    step="0.01"
                                                    min="0"
                                                    value={inst.client_value || ""}
                                                    onChange={(e) =>
                                                        updateInstallment(index, "client_value", e.target.value)
                                                    }
                                                    disabled={disabled || !canEditValues}
                                                    placeholder="Cliente"
                                                    className="financial-form-input"
                                                />
                                            </div>
                                            <div className="financial-form-installment-value">
                                                <input
                                                    type="number"
                                                    step="0.01"
                                                    min="0"
                                                    value={inst.received_value || ""}
                                                    onChange={(e) =>
                                                        updateInstallment(index, "received_value", e.target.value)
                                                    }
                                                    disabled={disabled || !canEditValues}
                                                    placeholder="Recebido"
                                                    className="financial-form-input"
                                                />
                                            </div>
                                        </>
                                    )}
                                    <button
                                        type="button"
                                        onClick={() => removeInstallment(index)}
                                        disabled={disabled || installments.length <= 1}
                                        className="financial-form-remove-btn"
                                        title="Remover parcela"
                                    >
                                        ✕
                                    </button>
                                </div>
                            ))}
                        </div>

                        {/* Totais */}
                        {showValues && (
                            <div className="financial-form-totals">
                                <div className="financial-form-total-item">
                                    <span>Total Cliente:</span>
                                    <strong>{formatCurrency(totalClientValue)}</strong>
                                </div>
                                <div className="financial-form-total-item">
                                    <span>Total Recebido:</span>
                                    <strong>{formatCurrency(totalReceivedValue)}</strong>
                                </div>
                            </div>
                        )}
                    </div>
                )}

                {/* Descrição */}
                <div className="financial-form-field">
                    <label>
                        Descrição
                        <span className="financial-form-optional">(opcional)</span>
                    </label>
                    <input
                        type="text"
                        value={description}
                        onChange={(e) => setDescription(e.target.value)}
                        disabled={disabled}
                        placeholder="Ex: Comissão padrão, Plano Premium, etc."
                        className="financial-form-input"
                    />
                </div>

                {/* Dica */}
                <p className="financial-form-tip">
                    💡 <strong>Dica:</strong> Use "Personalizado" para definir comissões com entrada + parcelas,
                    comum em planos de saúde. Os valores são opcionais.
                </p>
            </div>
        </div>
    );
}
