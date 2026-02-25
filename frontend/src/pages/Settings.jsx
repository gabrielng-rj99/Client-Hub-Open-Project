/*
 * Client Hub Open Project
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
import { useConfig } from "../contexts/ConfigContext";
import SecuritySettings from "../components/settings/SecuritySettings";
import RolesPermissions from "../components/settings/RolesPermissions";
import "./styles/Settings.css";

const GenderSelect = ({ section, fieldKey, value, onChange }) => (
    <select
        value={value}
        onChange={(e) => onChange(section, fieldKey, e.target.value)}
        className="gender-select"
    >
        <option value="M">Masculino (o)</option>
        <option value="F">Feminino (a)</option>
        <option value="E">Neutro (e)</option>
    </select>
);

export default function Settings({ token, apiUrl, user }) {
    const { config, updateSettings } = useConfig();

    const [formData, setFormData] = useState(config);
    const [brandingMessage, setBrandingMessage] = useState("");
    const [brandingError, setBrandingError] = useState("");
    const [brandingSaving, setBrandingSaving] = useState(false);
    const [labelsMessage, setLabelsMessage] = useState("");
    const [labelsError, setLabelsError] = useState("");
    const [labelsSaving, setLabelsSaving] = useState(false);
    const [dashboardMessage, setDashboardMessage] = useState("");
    const [dashboardError, setDashboardError] = useState("");
    const [dashboardSaving, setDashboardSaving] = useState(false);
    const [dashboardSettings, setDashboardSettings] = useState({
        show_birthdays: true,
        birthdays_days_ahead: 7,
        show_recent_activity: true,
        recent_activity_count: 10,
        show_statistics: true,
        show_expiring_contracts: true,
        expiring_days_ahead: 30,
        show_quick_actions: true,
    });

    // Active section tab
    const [activeSection, setActiveSection] = useState("branding");

    const canManageDashboard = user?.resources?.['settings']?.includes('update') || user?.role === 'root';
    const canManageSecurity = user?.resources?.['settings']?.includes('manage_security') || user?.role === 'root';
    const canManageRoles = user?.resources?.['roles']?.includes('read') || user?.role === 'root';

    // Sync form data when config loads (if loaded after mount)
    useEffect(() => {
        setFormData(config);
    }, [config]);

    const handleChange = (section, key, value) => {
        setFormData((prev) => ({
            ...prev,
            [section]: {
                ...prev[section],
                [key]: value,
            },
        }));
    };

    const handleKeyDown = (e, saveFunction) => {
        if (e.key === "Enter") {
            e.preventDefault();
            saveFunction();
        }
    };

    const handleSaveBranding = async () => {
        setBrandingMessage("");
        setBrandingError("");
        setBrandingSaving(true);
        try {
            // Check if useCustomLogo is truly enabled (handle both boolean and string)
            const isCustomLogoEnabled =
                formData.branding?.useCustomLogo === true ||
                formData.branding?.useCustomLogo === "true";

            // Only include logo URLs if useCustomLogo is true
            const brandingToSave = {
                appName: formData.branding?.appName || "",
                useCustomLogo: isCustomLogoEnabled,
            };

            // Only include logo URLs if custom logo is enabled
            if (isCustomLogoEnabled) {
                brandingToSave.logoWideUrl =
                    formData.branding?.logoWideUrl || "";
                brandingToSave.logoSquareUrl =
                    formData.branding?.logoSquareUrl || "";
            }

            const systemSettings = {
                branding: brandingToSave,
            };
            await updateSettings(systemSettings);

            setBrandingMessage("Branding salvo com sucesso!");
        } catch (err) {
            setBrandingError(
                err.message || "Erro ao salvar branding. Tente novamente.",
            );
        } finally {
            setBrandingSaving(false);
        }
    };

    const handleSaveLabels = async () => {
        setLabelsMessage("");
        setLabelsError("");
        setLabelsSaving(true);
        try {
            const systemSettings = {
                labels: formData.labels,
            };
            await updateSettings(systemSettings);

            setLabelsMessage("Rótulos salvos com sucesso!");
        } catch (err) {
            setLabelsError(
                err.message || "Erro ao salvar rótulos. Tente novamente.",
            );
        } finally {
            setLabelsSaving(false);
        }
    };

    const handleSaveDashboard = async () => {
        setDashboardMessage("");
        setDashboardError("");
        setDashboardSaving(true);
        try {
            // Save dashboard settings to system_settings
            const response = await fetch(`${apiUrl}/system-config/dashboard`, {
                method: "PUT",
                headers: {
                    "Content-Type": "application/json",
                    Authorization: `Bearer ${token}`,
                },
                body: JSON.stringify(dashboardSettings),
            });

            if (!response.ok) {
                throw new Error("Falha ao salvar configurações do dashboard");
            }

            // Recarregar as configurações do dashboard para garantir persistência
            const loadResponse = await fetch(
                `${apiUrl}/system-config/dashboard`,
                {
                    method: "GET",
                    headers: {
                        Authorization: `Bearer ${token}`,
                    },
                },
            );

            if (loadResponse.ok) {
                const data = await loadResponse.json();
                setDashboardSettings(data);
            }

            setDashboardMessage(
                "Configurações do Dashboard salvas com sucesso!",
            );
        } catch (err) {
            setDashboardError(
                err.message || "Erro ao salvar configurações. Tente novamente.",
            );
        } finally {
            setDashboardSaving(false);
        }
    };

    const handleDashboardChange = (key, value) => {
        setDashboardSettings((prev) => ({
            ...prev,
            [key]: value,
        }));
    };

    // Load dashboard settings lazily when the dashboard tab is active
    const [dashboardSettingsLoaded, setDashboardSettingsLoaded] = useState(false);
    useEffect(() => {
        if (activeSection !== "dashboard" || dashboardSettingsLoaded) return;
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
            loadDashboardSettings();
            setDashboardSettingsLoaded(true);
        }
    }, [activeSection, dashboardSettingsLoaded, token, apiUrl]);

    const handleSaveLabelsOriginal = async () => {
        setLabelsMessage("");
        setLabelsError("");
        setLabelsSaving(true);
        try {
            const systemSettings = {
                labels: formData.labels,
            };
            await updateSettings(systemSettings);

            setLabelsMessage("Rótulos salvos com sucesso!");
        } catch (err) {
            setLabelsError(
                err.message || "Erro ao salvar rótulos. Tente novamente.",
            );
        } finally {
            setLabelsSaving(false);
        }
    };

    const handleFileUpload = async (e, section, key) => {
        const file = e.target.files[0];
        if (!file) return;

        // Validation: Max 15MB
        if (file.size > 15 * 1024 * 1024) {
            setBrandingError("A imagem deve ter no máximo 15MB.");
            return;
        }

        // Validate file type
        const allowedTypes = [
            "image/jpeg",
            "image/png",
            "image/gif",
            "image/webp",
            "image/svg+xml",
        ];
        if (!allowedTypes.includes(file.type)) {
            setBrandingError(
                "Tipo de arquivo inválido. Permitidos: JPG, PNG, GIF, WEBP, SVG.",
            );
            return;
        }

        setBrandingError("");
        setBrandingSaving(true);

        try {
            // Upload file to server
            const formData = new FormData();
            formData.append("file", file);

            const response = await fetch(`${apiUrl}/upload`, {
                method: "POST",
                headers: {
                    Authorization: `Bearer ${token}`,
                },
                body: formData,
            });

            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.error || "Erro ao fazer upload");
            }

            const data = await response.json();

            // Use the server URL instead of base64
            handleChange(section, key, data.url);
            setBrandingMessage("Imagem carregada com sucesso!");
        } catch (err) {
            console.error("Upload error:", err);
            setBrandingError(err.message || "Erro ao fazer upload da imagem.");
        } finally {
            setBrandingSaving(false);
        }
    };

    return (
        <div className="settings-container">
            <div className="settings-header">
                <h1 className="settings-title">⚙️ Configurações do Sistema</h1>
            </div>

            {/* Main Navigation Tabs */}
            <div className="settings-main-tabs">
                <button
                    className={`settings-main-tab ${activeSection === "branding" ? "active" : ""}`}
                    onClick={() => setActiveSection("branding")}
                >
                    🎨 Marca e Identidade
                </button>
                <button
                    className={`settings-main-tab ${activeSection === "labels" ? "active" : ""}`}
                    onClick={() => setActiveSection("labels")}
                >
                    🏷️ Rótulos
                </button>
                {canManageDashboard && (
                    <button
                        className={`settings-main-tab ${activeSection === "dashboard" ? "active" : ""}`}
                        onClick={() => setActiveSection("dashboard")}
                    >
                        📊 Dashboard
                    </button>
                )}
                {canManageSecurity && (
                    <button
                        className={`settings-main-tab ${activeSection === "security" ? "active" : ""}`}
                        onClick={() => setActiveSection("security")}
                    >
                        🔒 Segurança
                    </button>
                )}
                {canManageRoles && (
                    <button
                        className={`settings-main-tab ${activeSection === "roles" ? "active" : ""}`}
                        onClick={() => setActiveSection("roles")}
                    >
                        👥 Papéis e Permissões
                    </button>
                )}
            </div>

            {/* Dashboard Settings Section */}
            {activeSection === "dashboard" && canManageDashboard && (
                <form
                    onSubmit={(e) => e.preventDefault()}
                    className="settings-form"
                >
                    <section className="settings-section">
                        <h2>📊 Configurações do Dashboard</h2>
                        <p className="section-description">
                            Configure quais widgets e informações serão exibidos
                            no painel principal.
                        </p>

                        {dashboardMessage && (
                            <div className="settings-success">
                                {dashboardMessage}
                            </div>
                        )}
                        {dashboardError && (
                            <div className="settings-error">
                                {dashboardError}
                            </div>
                        )}

                        {/* Aniversariantes */}
                        <div className="dashboard-settings-grid">
                            {/* Aniversariantes */}
                            <div className="settings-card">
                                <h3>🎂 Aniversariantes</h3>
                                <div className="card-content">
                                    <div className="switch-row">
                                        <label className="switch small">
                                            <input
                                                type="checkbox"
                                                checked={
                                                    dashboardSettings.show_birthdays
                                                }
                                                onChange={(e) =>
                                                    handleDashboardChange(
                                                        "show_birthdays",
                                                        e.target.checked,
                                                    )
                                                }
                                            />
                                            <span className="slider round"></span>
                                        </label>
                                        <span>Ativar Widget</span>
                                    </div>
                                    {dashboardSettings.show_birthdays && (
                                        <div className="compact-input-row">
                                            <label>Dias:</label>
                                            <input
                                                type="number"
                                                className="small-input"
                                                min="1"
                                                max="90"
                                                value={
                                                    dashboardSettings.birthdays_days_ahead
                                                }
                                                onChange={(e) =>
                                                    handleDashboardChange(
                                                        "birthdays_days_ahead",
                                                        parseInt(
                                                            e.target.value,
                                                        ) || 7,
                                                    )
                                                }
                                                onKeyDown={(e) =>
                                                    handleKeyDown(
                                                        e,
                                                        handleSaveDashboard,
                                                    )
                                                }
                                            />
                                        </div>
                                    )}
                                </div>
                            </div>

                            {/* Atividade Recente */}
                            <div className="settings-card">
                                <h3>📋 Atividade Recente</h3>
                                <div className="card-content">
                                    <div className="switch-row">
                                        <label className="switch small">
                                            <input
                                                type="checkbox"
                                                checked={
                                                    dashboardSettings.show_recent_activity
                                                }
                                                onChange={(e) =>
                                                    handleDashboardChange(
                                                        "show_recent_activity",
                                                        e.target.checked,
                                                    )
                                                }
                                            />
                                            <span className="slider round"></span>
                                        </label>
                                        <span>Ativar Widget</span>
                                    </div>
                                    {dashboardSettings.show_recent_activity && (
                                        <div className="compact-input-row">
                                            <label>Itens:</label>
                                            <input
                                                type="number"
                                                className="small-input"
                                                min="1"
                                                max="365"
                                                value={
                                                    dashboardSettings.recent_activity_count
                                                }
                                                onChange={(e) =>
                                                    handleDashboardChange(
                                                        "recent_activity_count",
                                                        parseInt(
                                                            e.target.value,
                                                        ) || 15,
                                                    )
                                                }
                                                onKeyDown={(e) =>
                                                    handleKeyDown(
                                                        e,
                                                        handleSaveDashboard,
                                                    )
                                                }
                                            />
                                        </div>
                                    )}
                                </div>
                            </div>

                            {/* Estatísticas */}
                            <div className="settings-card">
                                <h3>📈 Estatísticas</h3>
                                <div className="card-content">
                                    <div className="switch-row">
                                        <label className="switch small">
                                            <input
                                                type="checkbox"
                                                checked={
                                                    dashboardSettings.show_statistics
                                                }
                                                onChange={(e) =>
                                                    handleDashboardChange(
                                                        "show_statistics",
                                                        e.target.checked,
                                                    )
                                                }
                                            />
                                            <span className="slider round"></span>
                                        </label>
                                        <span>Exibir Cards de Totais</span>
                                    </div>
                                </div>
                            </div>

                            {/* Acordos Próximos ao Vencimento */}
                            <div className="settings-card">
                                <h3>
                                    ⏰ {config.labels?.contracts || "Contratos"}{" "}
                                    Próximos
                                </h3>
                                <div className="card-content">
                                    <div className="switch-row">
                                        <label className="switch small">
                                            <input
                                                type="checkbox"
                                                checked={
                                                    dashboardSettings.show_expiring_contracts
                                                }
                                                onChange={(e) =>
                                                    handleDashboardChange(
                                                        "show_expiring_contracts",
                                                        e.target.checked,
                                                    )
                                                }
                                            />
                                            <span className="slider round"></span>
                                        </label>
                                        <span>Exibir Widget</span>
                                    </div>
                                    {dashboardSettings.show_expiring_contracts && (
                                        <div className="compact-input-row">
                                            <label>Alerta:</label>
                                            <input
                                                type="number"
                                                className="small-input"
                                                min="1"
                                                max="365"
                                                value={
                                                    dashboardSettings.expiring_days_ahead
                                                }
                                                onChange={(e) =>
                                                    handleDashboardChange(
                                                        "expiring_days_ahead",
                                                        parseInt(
                                                            e.target.value,
                                                        ) || 30,
                                                    )
                                                }
                                                onKeyDown={(e) =>
                                                    handleKeyDown(
                                                        e,
                                                        handleSaveDashboard,
                                                    )
                                                }
                                            />
                                        </div>
                                    )}
                                </div>
                            </div>

                            {/* Ações Rápidas */}
                            <div className="settings-card">
                                <h3>⚡ Ações Rápidas</h3>
                                <div className="card-content">
                                    <div className="switch-row">
                                        <label className="switch small">
                                            <input
                                                type="checkbox"
                                                checked={
                                                    dashboardSettings.show_quick_actions
                                                }
                                                onChange={(e) =>
                                                    handleDashboardChange(
                                                        "show_quick_actions",
                                                        e.target.checked,
                                                    )
                                                }
                                            />
                                            <span className="slider round"></span>
                                        </label>
                                        <span>Botões de Ação</span>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <button
                            type="button"
                            onClick={handleSaveDashboard}
                            className="settings-save-btn"
                            disabled={dashboardSaving}
                        >
                            {dashboardSaving
                                ? "Salvando..."
                                : "Salvar Configurações do Dashboard"}
                        </button>
                    </section>
                </form>
            )}

            {/* Branding Section */}
            {activeSection === "branding" && (
                <form
                    onSubmit={(e) => e.preventDefault()}
                    className="settings-form"
                >
                    <section className="settings-section">
                        <h2>Marca e Identidade</h2>
                        {brandingMessage && (
                            <div className="settings-success">
                                {brandingMessage}
                            </div>
                        )}
                        {brandingError && (
                            <div className="settings-error">
                                {brandingError}
                            </div>
                        )}
                        <div className="form-group">
                            <label>Nome da Aplicação</label>
                            <input
                                type="text"
                                value={formData.branding?.appName || ""}
                                onChange={(e) =>
                                    handleChange(
                                        "branding",
                                        "appName",
                                        e.target.value,
                                    )
                                }
                                onKeyDown={(e) =>
                                    handleKeyDown(e, handleSaveBranding)
                                }
                            />
                        </div>

                        <div
                            className="form-group"
                            style={{
                                display: "flex",
                                alignItems: "center",
                                gap: "10px",
                                margin: "20px 0",
                            }}
                        >
                            <label className="switch">
                                <input
                                    type="checkbox"
                                    checked={
                                        formData.branding?.useCustomLogo ===
                                        true ||
                                        formData.branding?.useCustomLogo ===
                                        "true"
                                    }
                                    onChange={(e) =>
                                        handleChange(
                                            "branding",
                                            "useCustomLogo",
                                            e.target.checked,
                                        )
                                    }
                                />
                                <span className="slider round"></span>
                            </label>
                            <span>
                                Usar Logo Personalizado (Substitui o Nome em
                                Texto)
                            </span>
                        </div>

                        {(formData.branding?.useCustomLogo === true ||
                            formData.branding?.useCustomLogo === "true") && (
                                <>
                                    <div className="form-group">
                                        <label>
                                            Logo Horizontal (Barra Lateral Aberta)
                                        </label>
                                        <p className="form-hint">
                                            Tamanho recomendado: 230x60 pixels
                                        </p>
                                        <div className="image-upload-container">
                                            {formData.branding?.logoWideUrl && (
                                                <img
                                                    src={
                                                        formData.branding
                                                            .logoWideUrl
                                                    }
                                                    alt="Logo Wide Preview"
                                                    className="logo-preview wide"
                                                />
                                            )}
                                            <div className="upload-controls">
                                                <input
                                                    type="text"
                                                    placeholder="URL da imagem ou upload"
                                                    value={
                                                        formData.branding
                                                            ?.logoWideUrl || ""
                                                    }
                                                    onChange={(e) =>
                                                        handleChange(
                                                            "branding",
                                                            "logoWideUrl",
                                                            e.target.value,
                                                        )
                                                    }
                                                    onKeyDown={(e) =>
                                                        handleKeyDown(
                                                            e,
                                                            handleSaveBranding,
                                                        )
                                                    }
                                                />
                                                <input
                                                    type="file"
                                                    accept="image/*,.svg"
                                                    onChange={(e) =>
                                                        handleFileUpload(
                                                            e,
                                                            "branding",
                                                            "logoWideUrl",
                                                        )
                                                    }
                                                    className="file-input"
                                                />
                                            </div>
                                        </div>
                                    </div>
                                    <div className="form-group">
                                        <label>
                                            Ícone Quadrado (Barra Lateral
                                            Minimizada)
                                        </label>
                                        <p className="form-hint">
                                            Tamanho recomendado: 50x60 pixels
                                        </p>
                                        <div className="image-upload-container">
                                            {formData.branding?.logoSquareUrl && (
                                                <img
                                                    src={
                                                        formData.branding
                                                            .logoSquareUrl
                                                    }
                                                    alt="Icon Square Preview"
                                                    className="logo-preview square"
                                                />
                                            )}
                                            <div className="upload-controls">
                                                <input
                                                    type="text"
                                                    placeholder="URL da imagem ou upload"
                                                    value={
                                                        formData.branding
                                                            ?.logoSquareUrl || ""
                                                    }
                                                    onChange={(e) =>
                                                        handleChange(
                                                            "branding",
                                                            "logoSquareUrl",
                                                            e.target.value,
                                                        )
                                                    }
                                                    onKeyDown={(e) =>
                                                        handleKeyDown(
                                                            e,
                                                            handleSaveBranding,
                                                        )
                                                    }
                                                />
                                                <input
                                                    type="file"
                                                    accept="image/*,.svg"
                                                    onChange={(e) =>
                                                        handleFileUpload(
                                                            e,
                                                            "branding",
                                                            "logoSquareUrl",
                                                        )
                                                    }
                                                    className="file-input"
                                                />
                                            </div>
                                        </div>
                                    </div>
                                </>
                            )}
                        <div className="settings-actions">
                            <button
                                type="button"
                                onClick={handleSaveBranding}
                                disabled={brandingSaving}
                                className="save-button save-branding-button"
                            >
                                {brandingSaving
                                    ? "Salvando..."
                                    : "Salvar Branding"}
                            </button>
                        </div>
                    </section>
                </form>
            )}

            {/* Labels Section */}
            {activeSection === "labels" && (
                <form
                    onSubmit={(e) => e.preventDefault()}
                    className="settings-form"
                >
                    <section className="settings-section">
                        <h2>Personalização de Rótulos</h2>
                        <p className="section-description">
                            Personalize os nomes das entidades conforme o
                            vocabulário da sua organização.<br></br>
                            Use o seletor para ajustar o gênero
                            (Novo/Nova/Nove).
                            <br></br>
                        </p>
                        {labelsMessage && (
                            <div className="settings-success">
                                {labelsMessage}
                            </div>
                        )}
                        {labelsError && (
                            <div className="settings-error">{labelsError}</div>
                        )}

                        <div className="settings-actions">
                            <button
                                type="button"
                                onClick={handleSaveLabels}
                                disabled={labelsSaving}
                                className="save-button save-labels-button"
                            >
                                {labelsSaving
                                    ? "Salvando..."
                                    : "Salvar Rótulos"}
                            </button>
                        </div>

                        <div className="label-config-grid">
                            <div className="label-header-row">
                                <div className="col-client">Cliente</div>
                                <div className="col-gender">Gênero</div>
                                <div className="col-input">Singular</div>
                                <div className="col-input">Plural</div>
                            </div>

                            {[
                                {
                                    key: "user",
                                    pluralKey: "users",
                                    genderKey: "user_gender",
                                    label: "Usuário",
                                },
                                {
                                    key: "client",
                                    pluralKey: "clients",
                                    genderKey: "client_gender",
                                    label: "Cliente",
                                },
                                {
                                    key: "affiliate",
                                    pluralKey: "affiliates",
                                    genderKey: "affiliate_gender",
                                    label: "Afiliado",
                                },
                                {
                                    key: "contract",
                                    pluralKey: "contracts",
                                    genderKey: "contract_gender",
                                    label: "Contrato",
                                },
                                {
                                    key: "category",
                                    pluralKey: "categories",
                                    genderKey: "category_gender",
                                    label: "Categoria",
                                },
                                {
                                    key: "subcategory",
                                    pluralKey: "subcategories",
                                    genderKey: "subcategory_gender",
                                    label: "Subcategoria",
                                },
                            ].map((item) => (
                                <div key={item.key} className="label-row">
                                    <div className="col-client">
                                        {item.label}
                                    </div>
                                    <div className="col-gender">
                                        <GenderSelect
                                            section="labels"
                                            fieldKey={item.genderKey}
                                            value={
                                                formData.labels?.[
                                                item.genderKey
                                                ] || "M"
                                            }
                                            onChange={handleChange}
                                        />
                                    </div>
                                    <div className="col-input">
                                        <input
                                            type="text"
                                            value={
                                                formData.labels?.[item.key] ||
                                                ""
                                            }
                                            onChange={(e) =>
                                                handleChange(
                                                    "labels",
                                                    item.key,
                                                    e.target.value,
                                                )
                                            }
                                            onKeyDown={(e) =>
                                                handleKeyDown(
                                                    e,
                                                    handleSaveLabels,
                                                )
                                            }
                                            placeholder="Singular"
                                        />
                                    </div>
                                    <div className="col-input">
                                        <input
                                            type="text"
                                            value={
                                                formData.labels?.[
                                                item.pluralKey
                                                ] || ""
                                            }
                                            onChange={(e) =>
                                                handleChange(
                                                    "labels",
                                                    item.pluralKey,
                                                    e.target.value,
                                                )
                                            }
                                            onKeyDown={(e) =>
                                                handleKeyDown(
                                                    e,
                                                    handleSaveLabels,
                                                )
                                            }
                                        />
                                    </div>
                                </div>
                            ))}
                        </div>
                    </section>
                </form>
            )}

            {/* Security Section */}
            {activeSection === "security" && canManageSecurity && (
                <SecuritySettings token={token} apiUrl={apiUrl} />
            )}

            {/* Roles & Permissions Section */}
            {activeSection === "roles" && canManageRoles && (
                <RolesPermissions token={token} apiUrl={apiUrl} />
            )}
        </div>
    );
}
