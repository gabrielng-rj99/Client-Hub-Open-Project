import React from "react";
import { useConfig } from "../../contexts/ConfigContext";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { useNavigate, useLocation } from "react-router-dom";
import {
    faChartLine,
    faFileContract,
    faUserGroup,
    faTags,
    faMoneyBillWave,
    faUserGear,
    faSearchPlus,
    faCog,
    faPalette,
    faRightFromBracket,
} from "@fortawesome/free-solid-svg-icons";
import logoWideDefault from "../../assets/images/placeholder-230x60.png";
import logoSquareDefault from "../../assets/images/placeholder-50x60.png";
import "./Sidebar.css";

const Sidebar = ({ sidebarCollapsed, toggleSidebar, user, logout }) => {
    const { config = {} } = useConfig();
    const branding = config.branding || {};
    const labels = config.labels || {};

    // Normalize useCustomLogo to boolean
    const useCustomLogo =
        branding.useCustomLogo === true || branding.useCustomLogo === "true";
    const navigate = useNavigate();
    const location = useLocation();

    // Normalize pathname to determine active page
    // e.g. /dashboard -> dashboard
    const currentPath = location.pathname.substring(1) || "dashboard";

    const [logoWideError, setLogoWideError] = React.useState(false);
    const [logoSquareError, setLogoSquareError] = React.useState(false);

    return (
        <nav className={`app-nav ${sidebarCollapsed ? "collapsed" : ""}`}>
            {/* Branding Section - ACIMA (Above) the header/toggle */}
            <div
                className={`app-nav-branding ${useCustomLogo ? "visible" : "hidden"}`}
            >
                {/* Expanded State Logo */}
                <div className="sidebar-logo-expanded">
                    <img
                        src={
                            branding.logoWideUrl && !logoWideError
                                ? branding.logoWideUrl
                                : logoWideDefault
                        }
                        alt="Logo"
                        onError={() => setLogoWideError(true)}
                    />
                </div>
                {/* Collapsed State Logo (Icon) */}
                <div className="sidebar-logo-collapsed">
                    <img
                        src={
                            branding.logoSquareUrl && !logoSquareError
                                ? branding.logoSquareUrl
                                : logoSquareDefault
                        }
                        alt="Icon"
                        onError={() => setLogoSquareError(true)}
                    />
                </div>
            </div>

            <div className="app-nav-header">
                <h2 className="app-nav-title">
                    {branding.appName || "Client Hub"}
                </h2>
                <button onClick={toggleSidebar} className="app-nav-toggle">
                    {sidebarCollapsed ? "☰" : "←"}
                </button>
            </div>

            <div className="app-nav-items">
                <button
                    onClick={() => navigate("/dashboard")}
                    data-navigate={true}
                    className={`app-nav-button ${currentPath === "dashboard" ? "active" : ""}`}
                    title="Dashboard"
                >
                    <span className="app-nav-icon">
                        <FontAwesomeIcon icon={faChartLine} />
                    </span>
                    <span className="app-nav-text">Dashboard</span>
                </button>

                <button
                    onClick={() => navigate("/contracts")}
                    data-navigate={true}
                    className={`app-nav-button ${currentPath === "contracts" ? "active" : ""}`}
                    title={labels.contracts || "Contratos"}
                >
                    <span className="app-nav-icon">
                        <FontAwesomeIcon icon={faFileContract} />
                    </span>
                    <span className="app-nav-text">
                        {labels.contracts || "Contratos"}
                    </span>
                </button>

                <button
                    onClick={() => navigate("/clients")}
                    data-navigate={true}
                    className={`app-nav-button ${currentPath === "clients" ? "active" : ""}`}
                    title={labels.clients || "Clientes"}
                >
                    <span className="app-nav-icon">
                        <FontAwesomeIcon icon={faUserGroup} />
                    </span>
                    <span className="app-nav-text">
                        {labels.clients || "Clientes"}
                    </span>
                </button>

                <button
                    onClick={() => navigate("/categories")}
                    data-navigate={true}
                    className={`app-nav-button ${currentPath === "categories" ? "active" : ""}`}
                    title={labels?.categories || "Categorias"}
                >
                    <span className="app-nav-icon">
                        <FontAwesomeIcon icon={faTags} />
                    </span>
                    <span className="app-nav-text">
                        {labels?.categories || "Categorias"}
                    </span>
                </button>

                <button
                    onClick={() => navigate("/financial")}
                    data-navigate={true}
                    className={`app-nav-button ${currentPath === "financial" ? "active" : ""}`}
                    title="Financeiro"
                >
                    <span className="app-nav-icon">
                        <FontAwesomeIcon icon={faMoneyBillWave} />
                    </span>
                    <span className="app-nav-text">Financeiro</span>
                </button>

                {user?.resources?.['users']?.includes('read') && (
                    <button
                        onClick={() => navigate("/users")}
                        data-navigate={true}
                        className={`app-nav-button ${currentPath === "users" ? "active" : ""}`}
                        title={labels?.users || "Usuários"}
                    >
                        <span className="app-nav-icon">
                            <FontAwesomeIcon icon={faUserGear} />
                        </span>
                        <span className="app-nav-text">
                            {labels?.users || "Usuários"}
                        </span>
                    </button>
                )}

                {user?.resources?.['audit_logs']?.includes('read') && (
                    <button
                        onClick={() => navigate("/audit-logs")}
                        data-navigate={true}
                        className={`app-nav-button ${currentPath === "audit-logs" ? "active" : ""}`}
                        title="Logs"
                    >
                        <span className="app-nav-icon">
                            <FontAwesomeIcon icon={faSearchPlus} />
                        </span>
                        <span className="app-nav-text">Logs</span>
                    </button>
                )}

                <button
                    onClick={() => navigate("/appearance")}
                    data-navigate={true}
                    className={`app-nav-button ${currentPath === "appearance" ? "active" : ""}`}
                    title="Aparência"
                >
                    <span className="app-nav-icon">
                        <FontAwesomeIcon icon={faPalette} />
                    </span>
                    <span className="app-nav-text">Aparência</span>
                </button>

                {user?.resources?.['settings']?.includes('read') && (
                    <button
                        onClick={() => navigate("/settings")}
                        data-navigate={true}
                        className={`app-nav-button ${currentPath === "settings" ? "active" : ""}`}
                        title="Configurações"
                    >
                        <span className="app-nav-icon">
                            <FontAwesomeIcon icon={faCog} />
                        </span>
                        <span className="app-nav-text">Configurações</span>
                    </button>
                )}
            </div>

            <div className="app-nav-footer">
                <div className="app-nav-user-info">
                    <div className="app-nav-user-label">Usuário:</div>
                    <div className="user-name">{user.username}</div>
                    <div className="user-role">{user.role}</div>
                </div>
                <button
                    onClick={logout}
                    className="app-nav-logout-button"
                    title="Sair"
                >
                    <span className="app-nav-icon">
                        <FontAwesomeIcon icon={faRightFromBracket} />
                    </span>
                    <span className="app-nav-text">Sair</span>
                </button>
            </div>
        </nav>
    );
};

export default Sidebar;
