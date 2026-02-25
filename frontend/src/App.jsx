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

import React, { useState, useEffect, useRef } from "react";
import {
    BrowserRouter,
    Routes,
    Route,
    Navigate,
    Outlet,
} from "react-router-dom";
import Login from "./pages/Login";
import Dashboard from "./pages/Dashboard";
import Contracts from "./pages/Contracts";
import Clients from "./pages/Clients";
import Categories from "./pages/Categories";
import Financial from "./pages/Financial";
import Users from "./pages/Users";
import AuditLogs from "./pages/AuditLogs";
import Initialize from "./pages/Initialize";
import Settings from "./pages/Settings";
import Appearance from "./pages/Appearance";
import Sidebar from "./components/layout/Sidebar";
import LayoutManager from "./components/layout/LayoutManager";
import TopProgressBar from "./components/common/TopProgressBar";
import { ConfigProvider } from "./contexts/ConfigContext";
import { DataProvider } from "./contexts/DataContext";
import "./App.css";

const API_URL = import.meta.env.VITE_API_URL || "/api";

const ProtectedLayout = ({
    token,
    user,
    logout,
    sidebarCollapsed,
    toggleSidebar,
}) => {
    if (!token || !user) {
        return <Navigate to="/login" replace />;
    }

    return (
        <ConfigProvider apiUrl={API_URL} token={token} onTokenExpired={() => logout("Token inválido")}>
            <DataProvider
                token={token}
                apiUrl={API_URL}
                onTokenExpired={() => logout("Token inválido")}
            >
                <TopProgressBar />
                <LayoutManager />
                <div className="app-container">
                    <Sidebar
                        sidebarCollapsed={sidebarCollapsed}
                        toggleSidebar={toggleSidebar}
                        user={user}
                        logout={logout}
                    />
                    <main className="app-main">
                        <Outlet />
                    </main>
                </div>
            </DataProvider>
        </ConfigProvider>
    );
};

function App() {
    const [appReady, setAppReady] = useState(false);
    const [isInitializing, setIsInitializing] = useState(true);
    const [user, setUser] = useState(null);
    const [token, setToken] = useState(null);
    const [refreshToken, setRefreshToken] = useState(null);
    const [sidebarCollapsed, setSidebarCollapsed] = useState(
        window.innerWidth <= 768,
    );
    const [isMobileView, setIsMobileView] = useState(window.innerWidth <= 768);
    const refreshTimeoutRef = useRef(null);

    // Detect mobile/narrow screens and force sidebar collapsed permanently
    useEffect(() => {
        const handleResize = () => {
            const isMobile = window.innerWidth <= 768;
            setIsMobileView(isMobile);
            if (isMobile) {
                setSidebarCollapsed(true);
                document.documentElement.classList.add(
                    "mobile-forced-collapsed",
                );
            } else {
                document.documentElement.classList.remove(
                    "mobile-forced-collapsed",
                );
            }
        };

        handleResize();
        window.addEventListener("resize", handleResize);
        return () => window.removeEventListener("resize", handleResize);
    }, []);

    // Check if app is initialized
    // StrictMode-safe guard: prevent double-firing
    const initCheckStarted = useRef(false);
    useEffect(() => {
        if (initCheckStarted.current) return;
        initCheckStarted.current = true;
        const checkInitialization = async () => {
            try {
                const response = await fetch(`${API_URL}/initialize/status`);
                if (response.ok) {
                    const data = await response.json();
                    if (data.is_initialized) {
                        setAppReady(true);
                        setIsInitializing(false);
                    }
                }
            } catch (error) {
                setIsInitializing(true);
            }
        };

        checkInitialization();
    }, []);

    // Load saved session on mount
    useEffect(() => {
        try {
            const savedToken = localStorage.getItem("token");
            const savedRefreshToken = localStorage.getItem("refreshToken");
            const savedUser = localStorage.getItem("user");

            if (savedToken && savedRefreshToken && savedUser) {
                setToken(savedToken);
                setRefreshToken(savedRefreshToken);
                setUser(JSON.parse(savedUser));

                // Fetch fresh permissions to ensure they are up to date
                fetch(`${API_URL}/user/permissions`, {
                    headers: { "Authorization": `Bearer ${savedToken}` }
                })
                    .then(res => res.ok ? res.json() : null)
                    .then(permsData => {
                        if (permsData) {
                            const parsedUser = JSON.parse(savedUser);
                            parsedUser.permissions = permsData.permissions || {};
                            parsedUser.resources = permsData.resources || {};
                            setUser(parsedUser);
                            localStorage.setItem("user", JSON.stringify(parsedUser));
                        }
                    })
                    .catch(err => console.error("Failed to fetch fresh permissions:", err));
            }
        } catch (error) {
            console.error("Error loading session:", error);
            localStorage.clear();
        }
    }, []);

    const getTokenExpiration = (token) => {
        try {
            const payload = JSON.parse(atob(token.split(".")[1]));
            return payload.exp ? payload.exp * 1000 : null;
        } catch {
            return null;
        }
    };

    const scheduleTokenRefresh = (token, refreshToken) => {
        if (!token || !refreshToken) return;
        const exp = getTokenExpiration(token);
        if (!exp) return;
        const now = Date.now();

        if (exp - now < 30000) return;

        const msUntilRefresh = Math.max(exp - now - 2 * 60 * 1000, 60000);

        if (refreshTimeoutRef.current) {
            clearTimeout(refreshTimeoutRef.current);
        }

        refreshTimeoutRef.current = setTimeout(() => {
            renewAccessToken(refreshToken);
        }, msUntilRefresh);
    };

    const renewAccessToken = async (refreshToken) => {
        try {
            const response = await fetch(`${API_URL}/refresh-token`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ refresh_token: refreshToken }),
            });
            if (!response.ok) {
                throw new Error("Sessão expirada. Faça login novamente.");
            }
            const data = await response.json();
            const newToken = data.data.token;

            if (newToken && newToken !== token) {
                setToken(newToken);
                try {
                    localStorage.setItem("token", newToken);
                    localStorage.setItem("accessToken", newToken); // Sync for ConfigContext compatibility
                } catch (e) {
                    console.error("Error saving token:", e);
                }
            }
        } catch (err) {
            logout();
        }
    };

    const login = async (username, password) => {
        try {
            const response = await fetch(`${API_URL}/login`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ username, password }),
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || "Login falhou");
            }

            const data = await response.json();
            const userData = {
                id: data.data.user_id,
                username: data.data.username,
                role: data.data.role,
                permissions: {},
                resources: {},
            };

            try {
                const permsResponse = await fetch(`${API_URL}/user/permissions`, {
                    headers: { "Authorization": `Bearer ${data.data.token}` }
                });
                if (permsResponse.ok) {
                    const permsData = await permsResponse.json();
                    userData.permissions = permsData.permissions || {};
                    userData.resources = permsData.resources || {};
                }
            } catch (permError) {
                console.error("Failed to fetch permissions during login:", permError);
            }

            setToken(data.data.token);
            setRefreshToken(data.data.refresh_token);
            setUser(userData);

            try {
                localStorage.setItem("token", data.data.token);
                localStorage.setItem("refreshToken", data.data.refresh_token);
                localStorage.setItem("user", JSON.stringify(userData));
                localStorage.setItem("userRole", data.data.role || "user");
                localStorage.setItem("accessToken", data.data.token); // For ConfigContext compatibility
            } catch (e) {
                console.error("Error saving to localStorage:", e);
            }

            scheduleTokenRefresh(data.data.token, data.data.refresh_token);
        } catch (error) {
            if (
                error.message === "Failed to fetch" ||
                error.name === "TypeError"
            ) {
                throw new Error(
                    `Não foi possível conectar ao servidor. Verifique se o backend está rodando em ${API_URL}`,
                );
            }
            throw error;
        }
    };

    const logout = (errorMessage = null) => {
        // Notify backend of logout (fire-and-forget — don't block UI)
        const currentToken = localStorage.getItem("accessToken");
        if (currentToken && !(errorMessage && errorMessage.includes("Token inválido"))) {
            fetch(`${API_URL}/logout`, {
                method: "POST",
                headers: { Authorization: `Bearer ${currentToken}` },
            }).catch(() => { }); // Ignore errors — user is logging out regardless
        }

        setToken(null);
        setUser(null);
        setRefreshToken(null);

        if (
            errorMessage &&
            typeof errorMessage === "string" &&
            errorMessage.includes("Token inválido")
        ) {
            const url = new URL(window.location.href);
            url.searchParams.set("session_expired", "true");
            window.history.replaceState({}, "", url);
        }

        try {
            localStorage.removeItem("token");
            localStorage.removeItem("refreshToken");
            localStorage.removeItem("user");
            localStorage.removeItem("currentPage");
            localStorage.removeItem("userRole");
            localStorage.removeItem("accessToken");
            localStorage.removeItem("themeMode");
            localStorage.removeItem("accessibility");
        } catch (e) {
            console.error("Error clearing localStorage:", e);
        }

        if (refreshTimeoutRef.current) clearTimeout(refreshTimeoutRef.current);
    };

    const toggleSidebar = () => {
        // Prevent sidebar expansion on mobile/narrow screens
        if (isMobileView) {
            return;
        }
        setSidebarCollapsed(!sidebarCollapsed);
    };

    useEffect(() => {
        if (sidebarCollapsed) {
            document.documentElement.classList.add("collapsed");
        } else {
            document.documentElement.classList.remove("collapsed");
        }
    }, [sidebarCollapsed]);

    useEffect(() => {
        if (token && refreshToken) {
            const exp = getTokenExpiration(token);
            const now = Date.now();

            if (exp && exp - now > 60000) {
                scheduleTokenRefresh(token, refreshToken);
            } else if (exp && exp - now <= 60000 && exp - now > 0) {
                renewAccessToken(refreshToken);
            }
        }
        return () => {
            if (refreshTimeoutRef.current)
                clearTimeout(refreshTimeoutRef.current);
        };
    }, [token, refreshToken]);

    if (isInitializing) {
        return (
            <Initialize
                onInitializationComplete={() => {
                    setIsInitializing(false);
                    setAppReady(true);
                }}
            />
        );
    }

    return (
        <BrowserRouter>
            <Routes>
                <Route
                    path="/login"
                    element={
                        user ? (
                            <Navigate to="/dashboard" replace />
                        ) : (
                            <Login onLogin={login} />
                        )
                    }
                />

                <Route
                    element={
                        <ProtectedLayout
                            token={token}
                            user={user}
                            logout={logout}
                            sidebarCollapsed={sidebarCollapsed}
                            toggleSidebar={toggleSidebar}
                        />
                    }
                >
                    <Route
                        path="/"
                        element={<Navigate to="/dashboard" replace />}
                    />
                    <Route
                        path="/dashboard"
                        element={
                            <Dashboard
                                token={token}
                                apiUrl={API_URL}
                                onTokenExpired={() => logout("Token inválido")}
                            />
                        }
                    />
                    <Route
                        path="/contracts"
                        element={
                            <Contracts
                                token={token}
                                apiUrl={API_URL}
                                onTokenExpired={() => logout("Token inválido")}
                            />
                        }
                    />
                    <Route
                        path="/clients"
                        element={
                            <Clients
                                token={token}
                                apiUrl={API_URL}
                                onTokenExpired={() => logout("Token inválido")}
                            />
                        }
                    />
                    <Route
                        path="/categories"
                        element={
                            <Categories
                                token={token}
                                apiUrl={API_URL}
                                onTokenExpired={() => logout("Token inválido")}
                            />
                        }
                    />
                    <Route
                        path="/financial"
                        element={
                            <Financial
                                token={token}
                                apiUrl={API_URL}
                                onTokenExpired={() => logout("Token inválido")}
                            />
                        }
                    />
                    <Route
                        path="/users"
                        element={
                            <Users
                                token={token}
                                apiUrl={API_URL}
                                user={user}
                                onLogout={logout}
                                onTokenExpired={() => logout("Token inválido")}
                            />
                        }
                    />
                    <Route
                        path="/settings"
                        element={<Settings token={token} apiUrl={API_URL} user={user} />}
                    />
                    <Route
                        path="/appearance"
                        element={<Appearance token={token} apiUrl={API_URL} />}
                    />
                    <Route
                        path="/audit-logs"
                        element={
                            <AuditLogs
                                token={token}
                                apiUrl={API_URL}
                                user={user}
                                onTokenExpired={() => logout("Token inválido")}
                            />
                        }
                    />
                </Route>
            </Routes>

            {/* SVG Filters for Color Blindness */}
            <svg
                className="colorblind-filters"
                xmlns="http://www.w3.org/2000/svg"
            >
                <defs>
                    <filter id="protanopia-filter">
                        <feColorMatrix
                            in="SourceGraphic"
                            type="matrix"
                            values="0.567 0.433 0 0 0  0.558 0.442 0 0 0  0 0.242 0.758 0 0  0 0 0 1 0"
                        />
                    </filter>
                    <filter id="deuteranopia-filter">
                        <feColorMatrix
                            in="SourceGraphic"
                            type="matrix"
                            values="0.625 0.375 0 0 0  0.7 0.3 0 0 0  0 0.3 0.7 0 0  0 0 0 1 0"
                        />
                    </filter>
                    <filter id="tritanopia-filter">
                        <feColorMatrix
                            in="SourceGraphic"
                            type="matrix"
                            values="0.95 0.05 0 0 0  0 0.433 0.567 0 0  0 0.475 0.525 0 0  0 0 0 1 0"
                        />
                    </filter>
                </defs>
            </svg>
        </BrowserRouter>
    );
}

export default App;
