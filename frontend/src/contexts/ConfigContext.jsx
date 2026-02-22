/*
 * Client Hub Open Project
 * Copyright (C) 2025 Client Hub Contributors
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published
 * by the Free Software Foundation, either version 3 of the License, or
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

import React, {
    createContext,
    useContext,
    useState,
    useEffect,
    useCallback,
    useRef,
} from "react";
import { THEME_PRESETS } from "./themes/index.js";
import { themeApi } from "../api/themeApi.js";

const ConfigContext = createContext();

export const useConfig = () => {
    const context = useContext(ConfigContext);
    if (!context) {
        throw new Error("useConfig must be used within a ConfigProvider");
    }
    return context;
};

// Re-export THEME_PRESETS for backward compatibility
export { THEME_PRESETS };

// Colorblind-Safe Palettes
const COLORBLIND_SAFE = {
    protanopia: {
        light: {
            buttonPrimary: "#0077bb",
            buttonSecondary: "#cc6677",
            bgPage: "#f0f4f8",
            bgCard: "#ffffff",
            textPrimary: "#1a1a1a",
            textSecondary: "#4a4a4a",
            borderDefault: "#88ccee",
        },
        dark: {
            buttonPrimary: "#88ccee",
            buttonSecondary: "#ee99aa",
            bgPage: "#1a1a1a",
            bgCard: "#2d2d2d",
            textPrimary: "#f0f4f8",
            textSecondary: "#cccccc",
            borderDefault: "#0077bb",
        },
    },
    deuteranopia: {
        light: {
            buttonPrimary: "#0077bb",
            buttonSecondary: "#ee7733",
            bgPage: "#f0f4f8",
            bgCard: "#ffffff",
            textPrimary: "#1a1a1a",
            textSecondary: "#4a4a4a",
            borderDefault: "#88ccee",
        },
        dark: {
            buttonPrimary: "#88ccee",
            buttonSecondary: "#ff9955",
            bgPage: "#1a1a1a",
            bgCard: "#2d2d2d",
            textPrimary: "#f0f4f8",
            textSecondary: "#cccccc",
            borderDefault: "#0077bb",
        },
    },
    tritanopia: {
        light: {
            buttonPrimary: "#ee3377",
            buttonSecondary: "#009988",
            bgPage: "#f0f4f8",
            bgCard: "#ffffff",
            textPrimary: "#1a1a1a",
            textSecondary: "#4a4a4a",
            borderDefault: "#cc3366",
        },
        dark: {
            buttonPrimary: "#ee7799",
            buttonSecondary: "#33bbaa",
            bgPage: "#1a1a1a",
            bgCard: "#2d2d2d",
            textPrimary: "#f0f4f8",
            textSecondary: "#cccccc",
            borderDefault: "#bb2255",
        },
    },
};

// Default Settings
const defaultSettings = {
    branding: {
        appName: "Client Hub",
        logoUrl:
            "https://via.placeholder.com/150x50/3498db/ffffff?text=Client+Hub",
        minimizedIconUrl:
            "https://via.placeholder.com/32x32/3498db/ffffff?text=EH",
        logoWideUrl: "",
        logoSquareUrl: "",
        useCustomLogo: false,
    },
    labels: {
        client: "Cliente",
        client_gender: "M",
        clients: "Clientes",
        contract: "Contrato",
        contract_gender: "M",
        contracts: "Contratos",
        category: "Categoria",
        category_gender: "F",
        categories: "Categorias",
        subcategory: "Subcategoria",
        subcategory_gender: "F",
        subcategories: "Subcategorias",
        affiliate: "Afiliado",
        affiliate_gender: "M",
        affiliates: "Afiliados",
        user: "Usuário",
        user_gender: "M",
        users: "Usuários",
    },
    theme: {
        mode: "system",
        preset: "default",
        primaryColor: "#0284c7",
        secondaryColor: "#0369a1",
        backgroundColor: "#f0f9ff",
        surfaceColor: "#ffffff",
        textColor: "#0c4a6e",
        textSecondaryColor: "#475569",
        borderColor: "#7dd3fc",
        fontGeneral: "System",
        fontTitle: "System",
        fontTableTitle: "System",
    },
};

// Default label definitions
const defaultLabels = {
    client: "Cliente",
    client_gender: "M",
    clients: "Clientes",
    contract: "Contrato",
    contract_gender: "M",
    contracts: "Contratos",
    category: "Categoria",
    category_gender: "F",
    categories: "Categorias",
    subcategory: "Subcategoria",
    subcategory_gender: "F",
    subcategories: "Subcategorias",
    affiliate: "Afiliado",
    affiliate_gender: "M",
    affiliates: "Afiliados",
    user: "Usuário",
    user_gender: "M",
    users: "Usuários",
};

// Default accessibility preferences
const defaultAccessibility = {
    highContrast: false,
    colorBlindMode: "none",
    dyslexicFont: false,
};

export const ConfigProvider = ({ children, onTokenExpired }) => {
    // Load saved theme from localStorage to persist across refreshes
    const initialConfig = { ...defaultSettings };
    const savedTheme = localStorage.getItem("userTheme");
    if (savedTheme) {
        try {
            const parsed = JSON.parse(savedTheme);
            initialConfig.theme = { ...initialConfig.theme, ...parsed };
        } catch (e) {
            console.error("Error parsing saved theme", e);
        }
    }
    const [config, setConfig] = useState(initialConfig);
    const [loading, setLoading] = useState(true);

    // User theme settings from API
    const [userThemeSettings, setUserThemeSettings] = useState(null);
    const [canEditTheme, setCanEditTheme] = useState(false);
    const [themePermissions, setThemePermissions] = useState({
        usersCanEditTheme: false,
        adminsCanEditTheme: true,
    });
    const [themeSaving, setThemeSaving] = useState(false);
    // Consolidated theme data from API (avoids multiple requests)
    const [allowedThemes, setAllowedThemes] = useState([]);
    const [globalTheme, setGlobalTheme] = useState(null);

    // Theme mode, layout mode, font settings, and accessibility preferences
    const [themeMode, setThemeModeState] = useState("system");
    const [layoutMode, setLayoutModeState] = useState("standard");
    const [fontSettings, setFontSettings] = useState({
        general: "System",
        title: "System",
        tableTitle: "System",
        tableContent: "System",
    });
    const [accessibility, setAccessibilityState] =
        useState(defaultAccessibility);

    // Resolved theme mode (light/dark)
    const [resolvedMode, setResolvedMode] = useState("light");

    // Ref to always have the latest settings available for save operations (prevents stale state bugs)
    const latestSettingsRef = useRef({
        themeMode: "system",
        layoutMode: "standard",
        fontSettings: {
            general: "System",
            title: "System",
            tableTitle: "System",
            tableContent: "System",
        },
        accessibility: defaultAccessibility,
        userThemeSettings: null,
        preset: "default",
    });

    // Update ref whenever a state changes (as a secondary sync)
    useEffect(() => {
        latestSettingsRef.current = {
            ...latestSettingsRef.current,
            themeMode,
            layoutMode,
            fontSettings,
            accessibility,
            userThemeSettings,
            preset: config.theme?.preset || "default",
        };
    }, [
        themeMode,
        layoutMode,
        fontSettings,
        accessibility,
        userThemeSettings,
        config.theme?.preset,
    ]);

    // Fetch system settings from API
    const fetchSettings = useCallback(async () => {
        const token = localStorage.getItem("accessToken");
        if (!token) {
            return;
        }
        try {
            const res = await fetch("/api/settings", {
                headers: {
                    Authorization: `Bearer ${token}`,
                },
            });
            if (!res.ok) {
                if (res.status === 401) {
                    onTokenExpired?.();
                    return;
                }
                throw new Error("Fetch settings error");
            }

            const data = await res.json();

            const newConfig = { ...defaultSettings };

            // Helper function to parse values from the API
            // Backend stores everything as strings, so we need to convert back
            const parseValue = (value) => {
                if (value === "true") return true;
                if (value === "false") return false;
                // Try to parse as number if it looks like one
                if (/^-?\d+$/.test(value)) return parseInt(value, 10);
                if (/^-?\d+\.\d+$/.test(value)) return parseFloat(value);
                return value;
            };

            // Handle both map (object) and array responses
            const settingsToProcess = Array.isArray(data)
                ? data
                : Object.entries(data).map(([key, value]) => ({ key, value }));

            for (const item of settingsToProcess) {
                const parts = item.key.split(".");
                let obj = newConfig;

                for (let i = 0; i < parts.length - 1; i++) {
                    let val = parts[i];
                    if (!obj[val]) obj[val] = {};
                    obj = obj[val];
                }
                obj[parts[parts.length - 1]] = parseValue(item.value);
            }

            setConfig(newConfig);
        } catch (err) {
            console.error("Error fetching settings:", err);
        }
    }, []);

    // Fetch user theme settings from API
    const fetchUserTheme = useCallback(async () => {
        const token = localStorage.getItem("accessToken");
        if (!token) {
            // Fall back to localStorage if not logged in
            const savedMode = localStorage.getItem("themeMode");
            const savedLayout = localStorage.getItem("layoutMode");
            const savedFonts = localStorage.getItem("fontSettings");
            const savedAccessibility = localStorage.getItem("accessibility");
            if (savedMode) setThemeModeState(savedMode);
            if (savedLayout) setLayoutModeState(savedLayout);
            if (savedFonts) {
                try {
                    setFontSettings(JSON.parse(savedFonts));
                } catch (e) {
                    console.error("Error parsing saved fonts:", e);
                }
            }
            if (savedAccessibility) {
                try {
                    setAccessibilityState(JSON.parse(savedAccessibility));
                } catch (e) {
                    console.error("Error parsing saved accessibility:", e);
                }
            }
            return;
        }

        try {
            const response = await themeApi.getUserTheme("/api", token, () => {
                // Token expired - redirect to login
                onTokenExpired?.();
            });

            // Set permissions
            setCanEditTheme(response.can_edit || false);
            if (response.permissions) {
                setThemePermissions({
                    usersCanEditTheme:
                        response.permissions.users_can_edit_theme || false,
                    adminsCanEditTheme:
                        response.permissions.admins_can_edit_theme || true,
                });
            }

            // Set allowed themes from consolidated response
            if (response.allowed_themes) {
                setAllowedThemes(response.allowed_themes);
            }

            // Set global theme for root users from consolidated response
            if (response.global_theme) {
                setGlobalTheme(response.global_theme);
            }

            // Set user theme settings
            if (response.settings) {
                const settings = themeApi.apiToFrontend(response.settings);
                setUserThemeSettings(settings);
                setThemeModeState(settings.mode || "system");
                setLayoutModeState(settings.layoutMode || "standard");
                setFontSettings({
                    general: settings.fontGeneral || "System",
                    title: settings.fontTitle || "System",
                    tableTitle: settings.fontTableTitle || "System",
                    tableContent: settings.fontTableContent || "System",
                });
                setAccessibilityState({
                    highContrast: settings.highContrast || false,
                    colorBlindMode: settings.colorBlindMode || "none",
                    dyslexicFont: settings.dyslexicFont || false,
                });

                // Sync ref immediately
                latestSettingsRef.current = {
                    ...latestSettingsRef.current,
                    themeMode: settings.mode || "system",
                    layoutMode: settings.layoutMode || "standard",
                    fontSettings: {
                        general: settings.fontGeneral || "System",
                        title: settings.fontTitle || "System",
                        tableTitle: settings.fontTableTitle || "System",
                        tableContent: settings.fontTableContent || "System",
                    },
                    accessibility: {
                        highContrast: settings.highContrast || false,
                        colorBlindMode: settings.colorBlindMode || "none",
                        dyslexicFont: settings.dyslexicFont || false,
                    },
                    userThemeSettings: settings,
                    preset: settings.preset || "default",
                };

                // Update config theme if user has custom settings
                // Use explicit check for preset to avoid falsy string issues
                const newPreset =
                    settings.preset && settings.preset !== ""
                        ? settings.preset
                        : config.theme.preset;
                const updatedTheme = {
                    preset: newPreset,
                    primaryColor:
                        settings.primaryColor || config.theme.primaryColor,
                    secondaryColor:
                        settings.secondaryColor || config.theme.secondaryColor,
                    backgroundColor:
                        settings.backgroundColor ||
                        config.theme.backgroundColor,
                    surfaceColor:
                        settings.surfaceColor || config.theme.surfaceColor,
                    textColor: settings.textColor || config.theme.textColor,
                    textSecondaryColor:
                        settings.textSecondaryColor ||
                        config.theme.textSecondaryColor,
                    borderColor:
                        settings.borderColor || config.theme.borderColor,
                    fontGeneral:
                        settings.fontGeneral || config.theme.fontGeneral,
                    fontTitle: settings.fontTitle || config.theme.fontTitle,
                    fontTableTitle:
                        settings.fontTableTitle || config.theme.fontTableTitle,
                    fontTableContent:
                        settings.fontTableContent ||
                        config.theme.fontTableContent,
                };
                setConfig((prev) => ({
                    ...prev,
                    theme: updatedTheme,
                }));
                // Save to localStorage for persistence
                localStorage.setItem("userTheme", JSON.stringify(updatedTheme));
            } else {
                // No user settings - use localStorage as fallback
                const savedMode = localStorage.getItem("themeMode");
                const savedLayout = localStorage.getItem("layoutMode");
                const savedFonts = localStorage.getItem("fontSettings");
                const savedAccessibility =
                    localStorage.getItem("accessibility");
                if (savedMode) setThemeModeState(savedMode);
                if (savedLayout) setLayoutModeState(savedLayout);
                if (savedFonts) {
                    try {
                        setFontSettings(JSON.parse(savedFonts));
                    } catch (e) {
                        console.error("Error parsing saved fonts:", e);
                    }
                }
                if (savedAccessibility) {
                    try {
                        setAccessibilityState(JSON.parse(savedAccessibility));
                    } catch (e) {
                        console.error("Error parsing saved accessibility:", e);
                    }
                }
            }
        } catch (err) {
            console.error("Error fetching user theme:", err);
            // Fall back to localStorage
            const savedMode = localStorage.getItem("themeMode");
            const savedLayout = localStorage.getItem("layoutMode");
            const savedAccessibility = localStorage.getItem("accessibility");
            if (savedMode) setThemeModeState(savedMode);
            if (savedLayout) setLayoutModeState(savedLayout);
            if (savedAccessibility) {
                try {
                    setAccessibilityState(JSON.parse(savedAccessibility));
                } catch (e) {
                    console.error("Error parsing saved accessibility:", e);
                }
            }
        }
    }, []);

    // Save user theme settings to API
    const saveUserTheme = useCallback(
        async (themeSettings) => {
            const token = localStorage.getItem("accessToken");
            const userRole = localStorage.getItem("userRole") || "user";

            // Determine if user can save based on role and permissions
            // Root can always save, admin checks permissions, user checks permissions
            const canSave =
                userRole === "root" ||
                (userRole === "admin" && themePermissions.adminsCanEditTheme) ||
                (userRole === "user" && themePermissions.usersCanEditTheme) ||
                canEditTheme;

            if (!token || !canSave) {
                // Save to localStorage if not logged in or no permission
                if (themeSettings.mode) {
                    localStorage.setItem("themeMode", themeSettings.mode);
                }
                if (themeSettings.layoutMode) {
                    localStorage.setItem(
                        "layoutMode",
                        themeSettings.layoutMode,
                    );
                }
                if (
                    themeSettings.highContrast !== undefined ||
                    themeSettings.colorBlindMode !== undefined ||
                    themeSettings.dyslexicFont !== undefined
                ) {
                    localStorage.setItem(
                        "accessibility",
                        JSON.stringify({
                            highContrast: themeSettings.highContrast || false,
                            colorBlindMode:
                                themeSettings.colorBlindMode || "none",
                            dyslexicFont: themeSettings.dyslexicFont || false,
                        }),
                    );
                }
                return;
            }

            setThemeSaving(true);
            try {
                const apiSettings = themeApi.frontendToApi(themeSettings);
                await themeApi.updateUserTheme(
                    "/api",
                    token,
                    apiSettings,
                    () => {
                        console.error("Token expired while saving theme");
                    },
                );
                setUserThemeSettings(themeSettings);
            } catch (err) {
                console.error("Error saving user theme:", err);
                // Fall back to localStorage on error
                if (themeSettings.mode) {
                    localStorage.setItem("themeMode", themeSettings.mode);
                }
                if (themeSettings.layoutMode) {
                    localStorage.setItem(
                        "layoutMode",
                        themeSettings.layoutMode,
                    );
                }
                throw err;
            } finally {
                setThemeSaving(false);
            }
        },
        [canEditTheme, themePermissions],
    );

    // Save global theme settings (Admin/Root only)
    const saveGlobalTheme = useCallback(async (globalSettings) => {
        const token = localStorage.getItem("accessToken");
        if (!token) throw new Error("Token não encontrado");

        const apiSettings = themeApi.frontendToApi(globalSettings);

        const response = await fetch("/api/settings/global-theme", {
            method: "PUT",
            headers: {
                "Content-Type": "application/json",
                Authorization: `Bearer ${token}`,
            },
            body: JSON.stringify(apiSettings),
        });

        if (!response.ok) {
            const err = await response.json();
            throw new Error(err.error || "Erro ao salvar tema global");
        }
    }, []);

    // Save theme permissions (Root only)
    const saveThemePermissions = useCallback(
        async (usersCanEdit, adminsCanEdit) => {
            const token = localStorage.getItem("accessToken");
            if (!token) throw new Error("Token não encontrado");

            const response = await fetch("/api/settings/theme-permissions", {
                method: "PUT",
                headers: {
                    "Content-Type": "application/json",
                    Authorization: `Bearer ${token}`,
                },
                body: JSON.stringify({
                    users_can_edit_theme: usersCanEdit,
                    admins_can_edit_theme: adminsCanEdit,
                }),
            });

            if (!response.ok) {
                const err = await response.json();
                throw new Error(err.error || "Erro ao salvar permissões");
            }

            setThemePermissions({
                usersCanEditTheme: usersCanEdit,
                adminsCanEditTheme: adminsCanEdit,
            });
        },
        [],
    );
    // StrictMode-safe guard: prevent double-firing of initial settings load
    const settingsLoadStarted = useRef(false);
    useEffect(() => {
        if (settingsLoadStarted.current) return;
        settingsLoadStarted.current = true;
        const loadAllSettings = async () => {
            setLoading(true);
            try {
                await fetchSettings();
                await fetchUserTheme();
            } catch (err) {
                console.error("Error loading settings:", err);
            } finally {
                setLoading(false);
            }
        };
        loadAllSettings();
    }, [fetchSettings, fetchUserTheme]);

    // Update theme mode
    const setThemeMode = useCallback(
        async (mode) => {
            // Update ref synchronously first
            latestSettingsRef.current.themeMode = mode;

            // Update state and storage
            setThemeModeState(mode);
            localStorage.setItem("themeMode", mode);

            // Try to save to API if possible
            const token = localStorage.getItem("accessToken");
            if (token && canEditTheme) {
                try {
                    const current = latestSettingsRef.current;
                    const saveSettings = {
                        preset: current.preset,
                        mode: mode,
                        layoutMode: current.layoutMode,
                        primaryColor: config.theme?.primaryColor,
                        secondaryColor: config.theme?.secondaryColor,
                        backgroundColor: config.theme?.backgroundColor,
                        surfaceColor: config.theme?.surfaceColor,
                        textColor: config.theme?.textColor,
                        textSecondaryColor: config.theme?.textSecondaryColor,
                        borderColor: config.theme?.borderColor,
                        highContrast: current.accessibility.highContrast,
                        colorBlindMode: current.accessibility.colorBlindMode,
                        dyslexicFont: current.accessibility.dyslexicFont,
                        fontGeneral: current.fontSettings.general,
                        fontTitle: current.fontSettings.title,
                        fontTableTitle: current.fontSettings.tableTitle,
                        fontTableContent: current.fontSettings.tableContent,
                    };
                    await saveUserTheme(saveSettings);
                } catch (err) {
                    console.error("Error saving theme mode:", err);
                }
            }
        },
        [canEditTheme, config.theme, saveUserTheme],
    );

    // Update layout mode
    const setLayoutMode = useCallback(
        async (mode) => {
            if (
                mode !== "standard" &&
                mode !== "full" &&
                mode !== "centralized"
            )
                return;

            // Update ref synchronously first
            latestSettingsRef.current.layoutMode = mode;

            // Update state and storage
            setLayoutModeState(mode);
            localStorage.setItem("layoutMode", mode);

            const token = localStorage.getItem("accessToken");
            if (token && canEditTheme) {
                try {
                    const current = latestSettingsRef.current;
                    const saveSettings = {
                        preset: current.preset,
                        mode: current.themeMode,
                        layoutMode: mode,
                        primaryColor: config.theme?.primaryColor,
                        secondaryColor: config.theme?.secondaryColor,
                        backgroundColor: config.theme?.backgroundColor,
                        surfaceColor: config.theme?.surfaceColor,
                        textColor: config.theme?.textColor,
                        textSecondaryColor: config.theme?.textSecondaryColor,
                        borderColor: config.theme?.borderColor,
                        highContrast: current.accessibility.highContrast,
                        colorBlindMode: current.accessibility.colorBlindMode,
                        dyslexicFont: current.accessibility.dyslexicFont,
                        fontGeneral: current.fontSettings.general,
                        fontTitle: current.fontSettings.title,
                        fontTableTitle: current.fontSettings.tableTitle,
                        fontTableContent: current.fontSettings.tableContent,
                    };
                    await saveUserTheme(saveSettings);
                } catch (err) {
                    console.error("Error saving layout mode:", err);
                }
            }
        },
        [canEditTheme, config.theme, saveUserTheme],
    );

    // Update fonts
    const setFonts = useCallback(
        async (newFonts) => {
            // Update ref synchronously first
            const currentFonts = latestSettingsRef.current.fontSettings;
            const upFonts = { ...currentFonts, ...newFonts };
            latestSettingsRef.current.fontSettings = upFonts;

            // Update state and storage
            setFontSettings(upFonts);
            localStorage.setItem("fontSettings", JSON.stringify(upFonts));

            const token = localStorage.getItem("accessToken");
            if (token && canEditTheme) {
                const current = latestSettingsRef.current;
                const saveSettings = {
                    preset: current.preset,
                    mode: current.themeMode,
                    layoutMode: current.layoutMode,
                    primaryColor: config.theme?.primaryColor,
                    secondaryColor: config.theme?.secondaryColor,
                    backgroundColor: config.theme?.backgroundColor,
                    surfaceColor: config.theme?.surfaceColor,
                    textColor: config.theme?.textColor,
                    textSecondaryColor: config.theme?.textSecondaryColor,
                    borderColor: config.theme?.borderColor,
                    highContrast: current.accessibility.highContrast,
                    colorBlindMode: current.accessibility.colorBlindMode,
                    dyslexicFont: current.accessibility.dyslexicFont,
                    fontGeneral: upFonts.general,
                    fontTitle: upFonts.title,
                    fontTableTitle: upFonts.tableTitle,
                    fontTableContent: upFonts.tableContent,
                };
                saveUserTheme(saveSettings).catch((err) =>
                    console.error("Error saving font settings:", err),
                );
            }
        },
        [canEditTheme, config.theme, saveUserTheme],
    );

    // Apply layout class to root
    useEffect(() => {
        const root = document.documentElement;
        // Remove all layout classes first
        root.classList.remove(
            "layout-centralized",
            "layout-standard",
            "layout-full",
        );

        if (layoutMode === "full") {
            root.classList.add("layout-full");
        } else if (layoutMode === "centralized") {
            root.classList.add("layout-centralized");
        } else {
            // Default to standard (which is now the standard constrained layout)
            root.classList.add("layout-standard");
        }
    }, [layoutMode]);

    // Update accessibility settings
    const setAccessibility = useCallback(
        async (prefs) => {
            // Update ref synchronously first
            const currentAccessibility =
                latestSettingsRef.current.accessibility;
            const newPrefs = { ...currentAccessibility, ...prefs };
            latestSettingsRef.current.accessibility = newPrefs;

            // Update state and storage
            setAccessibilityState(newPrefs);
            localStorage.setItem("accessibility", JSON.stringify(newPrefs)); // Always save locally

            // Try to save to API if possible
            const token = localStorage.getItem("accessToken");
            if (token && canEditTheme) {
                const current = latestSettingsRef.current;
                const saveSettings = {
                    preset: current.preset,
                    mode: current.themeMode,
                    layoutMode: current.layoutMode,
                    primaryColor: config.theme?.primaryColor,
                    secondaryColor: config.theme?.secondaryColor,
                    backgroundColor: config.theme?.backgroundColor,
                    surfaceColor: config.theme?.surfaceColor,
                    textColor: config.theme?.textColor,
                    textSecondaryColor: config.theme?.textSecondaryColor,
                    borderColor: config.theme?.borderColor,
                    highContrast: newPrefs.highContrast,
                    colorBlindMode: newPrefs.colorBlindMode,
                    dyslexicFont: newPrefs.dyslexicFont,
                    fontGeneral: current.fontSettings.general,
                    fontTitle: current.fontSettings.title,
                    fontTableTitle: current.fontSettings.tableTitle,
                    fontTableContent: current.fontSettings.tableContent,
                };
                saveUserTheme(saveSettings).catch((err) =>
                    console.error("Error saving accessibility settings:", err),
                );
            }
        },
        [canEditTheme, config.theme, saveUserTheme],
    );

    // Check system theme preference
    const checkSystemTheme = () => {
        if (
            typeof window === "undefined" ||
            typeof window.matchMedia !== "function"
        ) {
            return "light";
        }

        const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
        const prefersDark =
            mediaQuery && typeof mediaQuery.matches === "boolean"
                ? mediaQuery.matches
                : false;

        return prefersDark ? "dark" : "light";
    };

    const updateResolvedMode = useCallback(() => {
        if (themeMode === "system") {
            setResolvedMode(checkSystemTheme());
        } else {
            setResolvedMode(themeMode);
        }
    }, [themeMode]);

    // Listen for system theme changes
    useEffect(() => {
        updateResolvedMode();
        if (
            typeof window === "undefined" ||
            typeof window.matchMedia !== "function"
        ) {
            return undefined;
        }

        const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
        if (!mediaQuery || typeof mediaQuery.addEventListener !== "function") {
            return undefined;
        }

        const handler = (e) => {
            if (themeMode === "system") {
                setResolvedMode(e.matches ? "dark" : "light");
            }
        };

        mediaQuery.addEventListener("change", handler);
        return () => mediaQuery.removeEventListener("change", handler);
    }, [themeMode, updateResolvedMode]);

    // Apply theme colors to CSS variables
    useEffect(() => {
        // Only apply theme if config is loaded and has valid theme data
        if (!config || loading || !config.theme) {
            return;
        }

        updateResolvedMode();
        const root = document.documentElement;

        const preset = config.theme.preset || "default";
        const primaryColor = config.theme.primaryColor;
        const secondaryColor = config.theme.secondaryColor;
        const backgroundColor = config.theme.backgroundColor;
        const surfaceColor = config.theme.surfaceColor;
        const textColor = config.theme.textColor;
        const textSecondaryColor = config.theme.textSecondaryColor;
        const borderColor = config.theme.borderColor;

        let themeColors;

        // Check for accessibility overrides first
        if (
            accessibility.colorBlindMode &&
            accessibility.colorBlindMode !== "none"
        ) {
            const colorblindPalette =
                COLORBLIND_SAFE[accessibility.colorBlindMode];
            themeColors =
                resolvedMode === "dark"
                    ? colorblindPalette.dark
                    : colorblindPalette.light;
        } else if (preset === "custom") {
            // Custom theme - create distinct light/dark variants from custom colors
            const customLight = {
                buttonPrimary: primaryColor || "#0284c7",
                buttonSecondary: secondaryColor || "#0369a1",
                bgPage: backgroundColor || "#f0f9ff",
                bgCard: surfaceColor || "#ffffff",
                textPrimary: textColor || "#0c4a6e",
                textSecondary: textSecondaryColor || "#475569",
                borderDefault: borderColor || "#7dd3fc",
            };

            // Create a properly contrasted dark variant
            const customDark = {
                buttonPrimary: lightenColor(primaryColor || "#0284c7", 15),
                buttonSecondary: lightenColor(secondaryColor || "#0369a1", 10),
                bgPage: darkenColor(backgroundColor || "#f0f9ff", 85),
                bgCard: darkenColor(surfaceColor || "#ffffff", 75),
                textPrimary: "#e0f2fe",
                textSecondary: "#94a3b8",
                borderDefault: darkenColor(borderColor || "#7dd3fc", 40),
            };
            themeColors = resolvedMode === "dark" ? customDark : customLight;
        } else if (preset && THEME_PRESETS[preset]) {
            // Preset theme
            themeColors =
                resolvedMode === "dark"
                    ? THEME_PRESETS[preset].dark
                    : THEME_PRESETS[preset].light;
        } else {
            // Fallback to default
            themeColors =
                resolvedMode === "dark"
                    ? THEME_PRESETS.default.dark
                    : THEME_PRESETS.default.light;
        }
        // Helper to get brightness (simplified)
        const getBrightness = (hex) => {
            if (!hex) return 0;
            const c = hex.replace("#", "");
            if (c.length !== 6) return 0;
            const rgb = parseInt(c, 16);
            const r = (rgb >> 16) & 0xff;
            const g = (rgb >> 8) & 0xff;
            const b = (rgb >> 0) & 0xff;
            return 0.2126 * r + 0.7152 * g + 0.0722 * b;
        };

        // Use text colors from theme
        let primaryTextColor = themeColors.textPrimary || "#222222";
        let secondaryTextColor = themeColors.textSecondary || "#444444";

        // Apply Mode Class
        if (resolvedMode === "dark") {
            root.classList.add("dark-mode");
            root.classList.remove("light-mode");
        } else {
            root.classList.add("light-mode");
            root.classList.remove("dark-mode");
        }

        // Safety check - ensure themeColors is defined
        if (!themeColors) {
            console.error("Theme colors not defined, using default");
            themeColors = THEME_PRESETS.default.light;
        }

        // Apply CSS variables (matching project's existing CSS variable names)
        root.style.setProperty(
            "--primary-color",
            themeColors.buttonPrimary || "#0284c7",
        );
        root.style.setProperty(
            "--secondary-color",
            themeColors.buttonSecondary || "#0369a1",
        );
        root.style.setProperty("--bg-color", themeColors.bgPage || "#f0f9ff");
        root.style.setProperty("--content-bg", themeColors.bgCard || "#ffffff");
        root.style.setProperty("--primary-text-color", primaryTextColor);
        root.style.setProperty("--secondary-text-color", secondaryTextColor);
        root.style.setProperty(
            "--border-color",
            themeColors.borderDefault || "#7dd3fc",
        );

        // Apply nav text color (falls back to secondaryTextColor if not defined)
        root.style.setProperty(
            "--nav-text-color",
            themeColors.textNav || secondaryTextColor,
        );

        // Apply app background color (falls back to secondary color if not defined)
        root.style.setProperty(
            "--app-bg-color",
            themeColors.bgNavbar || themeColors.buttonSecondary || "#0369a1",
        );

        // Apply page title color (falls back to text color if not defined)
        root.style.setProperty(
            "--page-title-color",
            themeColors.textTitle || themeColors.textPrimary || "#0369a1",
        );

        // Apply active nav text color (falls back to primaryTextColor if not defined)
        root.style.setProperty(
            "--active-nav-text-color",
            themeColors.textNavActive || primaryTextColor,
        );

        // Generate hover colors
        root.style.setProperty(
            "--primary-dark",
            darkenColor(themeColors.buttonPrimary, 10),
        );
        root.style.setProperty(
            "--secondary-light",
            darkenColor(themeColors.buttonSecondary, 10),
        );
    }, [config, resolvedMode, themeMode, accessibility, updateResolvedMode]);

    // Apply accessibility classes and attributes
    useEffect(() => {
        const root = document.documentElement;

        // High contrast filter
        if (accessibility.highContrast) {
            root.classList.add("high-contrast-filter");
        } else {
            root.classList.remove("high-contrast-filter");
        }

        // Apply Fonts
        const getFontStack = (fontName) => {
            switch (fontName) {
                // Accessibility
                case "OpenDyslexic 3":
                    return '"OpenDyslexic3", "Open Dyslexic", "Comic Sans MS", "Chalkboard SE", sans-serif';
                case "OpenDyslexic":
                    return '"OpenDyslexic", "Comic Sans MS", "Chalkboard SE", sans-serif';
                case "OpenDyslexic Mono":
                    return '"OpenDyslexicMono", monospace';

                // Web Fonts (Open Source) - Stacks
                case "Inter":
                    return 'Inter, "DejaVu Sans", "Liberation Sans", sans-serif';
                case "Roboto":
                    return 'Roboto, "DejaVu Sans", "Liberation Sans", sans-serif';
                case "Lato":
                    return 'Lato, "DejaVu Sans", "Liberation Sans", sans-serif';
                case "Montserrat":
                    return 'Montserrat, "DejaVu Sans", "Liberation Sans", sans-serif';
                case "Fira Code":
                    return '"Fira Code", "Fira Mono", "DejaVu Sans Mono", monospace';

                // Legacy / Windows fallbacks (mapped to closest Rep) or Keep for custom inputs
                case "Arial":
                    return 'Arial, "Liberation Sans", "DejaVu Sans", sans-serif';
                case "Times New Roman":
                    return '"Times New Roman", "Liberation Serif", "DejaVu Serif", serif';

                default:
                    if (!fontName || fontName === "System") {
                        return '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol"';
                    }
                    // Custom / Dynamic Web Font stack
                    return `"${fontName}", -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif`;
            }
        };

        if (accessibility.dyslexicFont) {
            root.style.setProperty(
                "--font-general",
                getFontStack("OpenDyslexic"),
            );
            root.style.setProperty(
                "--font-title",
                getFontStack("OpenDyslexic 3"),
            );
            root.style.setProperty(
                "--font-table-title",
                getFontStack("OpenDyslexic 3"),
            );
        } else {
            root.style.setProperty(
                "--font-general",
                getFontStack(fontSettings.general),
            );
            root.style.setProperty(
                "--font-title",
                getFontStack(fontSettings.title),
            );
            root.style.setProperty(
                "--font-table-title",
                getFontStack(fontSettings.tableTitle),
            );
        }

        const allSelectedFonts = new Set([
            fontSettings.general,
            fontSettings.title,
            fontSettings.tableTitle,
            fontSettings.tableContent,
        ]);

        // Dynamically load Google Fonts
        const CURATED_GOOGLE_FONTS = [
            "Inter",
            "Roboto",
            "Open Sans",
            "Lato",
            "Montserrat",
            "Poppins",
            "Fira Code",
            "Source Sans Pro",
            "Oswald",
            "Raleway",
            "Merriweather",
            "Nunito",
            "Atkinson Hyperlegible",
            "Lexend",
        ];

        const fontsToLoad = new Set([
            ...CURATED_GOOGLE_FONTS,
            fontSettings.general,
            fontSettings.title,
            fontSettings.tableTitle,
            fontSettings.tableContent,
        ]);

        // Fonts that are expected to be LOCAL
        const systemFonts = new Set([
            "System",
            "DejaVu Sans",
            "Liberation Sans",
            "Ubuntu",
            "DejaVu Serif",
            "Liberation Serif",
            "DejaVu Sans Mono",
            "Liberation Mono",
            "Arial",
            "Helvetica",
            "Verdana",
            "Tahoma",
            "Trebuchet MS",
            "Times New Roman",
            "Courier New",
            // Local Accessibility Fonts (now loaded via fonts.css)
            "OpenDyslexic",
            "OpenDyslexic 3",
            "OpenDyslexic Mono",
        ]);

        const googleFonts = Array.from(fontsToLoad).filter(
            (f) => !systemFonts.has(f) && f && f !== "System",
        );

        if (googleFonts.length > 0) {
            const linkId = "dynamic-web-fonts";
            let link = document.getElementById(linkId);
            if (!link) {
                link = document.createElement("link");
                link.id = linkId;
                link.rel = "stylesheet";
                document.head.appendChild(link);
            }

            // Build URL: https://fonts.googleapis.com/css2?family=Roboto:wght@300;400;500;700&family=Open+Sans:wght@400;600&display=swap
            const families = googleFonts
                .map(
                    (font) =>
                        `family=${font.replace(/\s+/g, "+")}:wght@300;400;500;600;700`,
                )
                .join("&");
            link.href = `https://fonts.googleapis.com/css2?${families}&display=swap`;
        }

        // Color blind mode

        // Color blind mode
        if (
            accessibility.colorBlindMode &&
            accessibility.colorBlindMode !== "none"
        ) {
            root.classList.add(`colorblind-${accessibility.colorBlindMode}`);
        } else {
            // Remove all colorblind classes
            root.classList.remove(
                "colorblind-protanopia",
                "colorblind-deuteranopia",
                "colorblind-tritanopia",
            );
        }
    }, [accessibility, fontSettings]);

    // Helper function to darken a color
    function darkenColor(hex, percent) {
        if (!hex) return hex;
        const c = hex.replace("#", "");
        if (c.length !== 6) return hex;
        const num = parseInt(c, 16);
        const amt = Math.round(2.55 * percent);
        let r = (num >> 16) - amt;
        let g = ((num >> 8) & 0x00ff) - amt;
        let b = (num & 0x0000ff) - amt;
        r = Math.max(0, r);
        g = Math.max(0, g);
        b = Math.max(0, b);
        return (
            "#" +
            (
                0x1000000 +
                (r < 255 ? (r < 1 ? 0 : r) : 255) * 0x10000 +
                (g < 255 ? (g < 1 ? 0 : g) : 255) * 0x100 +
                (b < 255 ? (b < 1 ? 0 : b) : 255)
            )
                .toString(16)
                .slice(1)
        );
    }

    // Helper function to lighten a color
    function lightenColor(hex, percent) {
        if (!hex) return hex;
        const c = hex.replace("#", "");
        if (c.length !== 6) return hex;
        const num = parseInt(c, 16);
        const amt = Math.round(2.55 * percent);
        let r = (num >> 16) + amt;
        let g = ((num >> 8) & 0x00ff) + amt;
        let b = (num & 0x0000ff) + amt;
        r = Math.min(255, r);
        g = Math.min(255, g);
        b = Math.min(255, b);
        return (
            "#" +
            (
                0x1000000 +
                (r < 255 ? (r < 1 ? 0 : r) : 255) * 0x10000 +
                (g < 255 ? (g < 1 ? 0 : g) : 255) * 0x100 +
                (b < 255 ? (b < 1 ? 0 : b) : 255)
            )
                .toString(16)
                .slice(1)
        );
    }

    // Update system settings (branding, labels, global theme)
    const updateSettings = async (newSettings) => {
        // Update local state
        setConfig((prev) => ({
            ...prev,
            ...newSettings,
        }));

        // If theme settings are being updated, also save user theme
        if (newSettings.theme && canEditTheme) {
            try {
                const themeSettings = {
                    preset:
                        newSettings.theme.preset ||
                        config.theme?.preset ||
                        "default",
                    mode: themeMode,
                    primaryColor: newSettings.theme.primaryColor || null,
                    secondaryColor: newSettings.theme.secondaryColor || null,
                    backgroundColor: newSettings.theme.backgroundColor || null,
                    surfaceColor: newSettings.theme.surfaceColor || null,
                    textColor: newSettings.theme.textColor || null,
                    textSecondaryColor:
                        newSettings.theme.textSecondaryColor || null,
                    borderColor: newSettings.theme.borderColor || null,
                    highContrast: accessibility.highContrast,
                    colorBlindMode: accessibility.colorBlindMode,
                    dyslexicFont: accessibility.dyslexicFont,
                    focusIndicator: accessibility.focusIndicator,
                };
                await saveUserTheme(themeSettings);
            } catch (err) {
                console.error("Error saving user theme:", err);
            }
        }

        // Convert nested settings to flat key-value pairs for system settings
        const flatSettings = {};
        const flattenObject = (obj, prefix = "") => {
            for (const key in obj) {
                const fullKey = prefix ? `${prefix}.${key}` : key;
                if (
                    typeof obj[key] === "object" &&
                    obj[key] !== null &&
                    !Array.isArray(obj[key])
                ) {
                    flattenObject(obj[key], fullKey);
                } else {
                    // Backend espera map[string]string, então convertemos valor para string
                    const value = obj[key];
                    flatSettings[fullKey] =
                        value === null || value === undefined
                            ? ""
                            : String(value);
                }
            }
        };

        // Only flatten non-theme settings for system settings API
        const systemSettings = { ...newSettings };
        delete systemSettings.theme; // Theme is handled separately per user
        flattenObject(systemSettings);

        // Save to API (system settings)
        const token = localStorage.getItem("accessToken");
        if (!token) return;

        try {
            const res = await fetch("/api/settings", {
                method: "PUT",
                headers: {
                    "Content-Type": "application/json",
                    Authorization: `Bearer ${token}`,
                },
                body: JSON.stringify({ settings: flatSettings }),
            });

            // Não recarregar configurações após salvar - o estado local já foi atualizado
            // Isso evita quebra temporária do tema durante o recarregamento
            if (!res.ok) throw new Error("Failed to save settings");
        } catch (err) {
            console.error("Error saving settings:", err);
        }
    };

    // Save only theme settings to user API
    const saveThemeSettings = useCallback(
        async (themeData) => {
            const userRole = localStorage.getItem("userRole") || "user";

            // Determine if user can save based on role and permissions
            // Root can always save, admin checks permissions, user checks permissions
            const canSave =
                userRole === "root" ||
                (userRole === "admin" && themePermissions.adminsCanEditTheme) ||
                (userRole === "user" && themePermissions.usersCanEditTheme) ||
                canEditTheme;

            if (!canSave) {
                console.warn("User does not have permission to edit theme");
                return;
            }

            const current = latestSettingsRef.current;
            const themeSettings = {
                preset: themeData.preset || current.preset || "default",
                mode: themeData.mode || current.themeMode,
                layoutMode:
                    themeData.layoutMode || current.layoutMode || "standard",
                primaryColor:
                    themeData.primaryColor !== undefined
                        ? themeData.primaryColor
                        : config.theme?.primaryColor || null,
                secondaryColor:
                    themeData.secondaryColor !== undefined
                        ? themeData.secondaryColor
                        : config.theme?.secondaryColor || null,
                backgroundColor:
                    themeData.backgroundColor !== undefined
                        ? themeData.backgroundColor
                        : config.theme?.backgroundColor || null,
                surfaceColor:
                    themeData.surfaceColor !== undefined
                        ? themeData.surfaceColor
                        : config.theme?.surfaceColor || null,
                textColor:
                    themeData.textColor !== undefined
                        ? themeData.textColor
                        : config.theme?.textColor || null,
                textSecondaryColor:
                    themeData.textSecondaryColor !== undefined
                        ? themeData.textSecondaryColor
                        : config.theme?.textSecondaryColor || null,
                borderColor:
                    themeData.borderColor !== undefined
                        ? themeData.borderColor
                        : config.theme?.borderColor || null,

                // ALWAYS use the latest accessibility and font settings from global state/ref,
                // ignoring whatever might be in themeData (which often contains stale copies)
                highContrast: current.accessibility.highContrast,
                colorBlindMode: current.accessibility.colorBlindMode,
                dyslexicFont: current.accessibility.dyslexicFont,

                fontGeneral: current.fontSettings.general,
                fontTitle: current.fontSettings.title,
                fontTableTitle: current.fontSettings.tableTitle,
                fontTableContent: current.fontSettings.tableContent,
            };

            try {
                await saveUserTheme(themeSettings);

                // Update local config
                const updatedTheme = {
                    preset: themeSettings.preset,
                    primaryColor: themeSettings.primaryColor,
                    secondaryColor: themeSettings.secondaryColor,
                    backgroundColor: themeSettings.backgroundColor,
                    surfaceColor: themeSettings.surfaceColor,
                    textColor: themeSettings.textColor,
                    textSecondaryColor: themeSettings.textSecondaryColor,
                    borderColor: themeSettings.borderColor,
                };
                setConfig((prev) => ({
                    ...prev,
                    theme: updatedTheme,
                }));
                // Save to localStorage for persistence
                localStorage.setItem("userTheme", JSON.stringify(updatedTheme));
            } catch (err) {
                console.error("Error saving theme settings:", err);
                throw err;
            }
        },
        [canEditTheme, themePermissions, config.theme, saveUserTheme],
    );

    // Persistent filter state
    const getPersistentFilter = (key) => {
        try {
            const saved = localStorage.getItem(`filter_${key}`);
            return saved ? JSON.parse(saved) : null;
        } catch {
            return null;
        }
    };

    const setPersistentFilter = (key, value) => {
        try {
            localStorage.setItem(`filter_${key}`, JSON.stringify(value));
        } catch (err) {
            console.error("Error saving filter:", err);
        }
    };

    // Helper to get label with fallback
    const getLabel = (key) => {
        const value = config?.labels?.[key] || defaultLabels[key] || key;
        return value;
    };

    // Helper to get gender-specific text
    const getGenderHelpers = (key) => {
        let gender = config?.labels?.[`${key}_gender`] || "M";

        // Singular label
        const value = getLabel(key);

        if (gender === "M") {
            return {
                label: value,
                article: "o",
                articleUpper: "O",
                new: "Novo",
                none: "Nenhum",
                found: "Encontrado",
                all: "Todos",
                active: "Ativo",
                inactive: "Inativo",
                archived: "Arquivado",
            };
        } else if (gender === "F") {
            return {
                label: value,
                article: "a",
                articleUpper: "A",
                new: "Nova",
                none: "Nenhuma",
                found: "Encontrada",
                all: "Todas",
                active: "Ativa",
                inactive: "Inativa",
                archived: "Arquivada",
            };
        } else {
            return {
                label: value,
                article: "e",
                articleUpper: "E",
                new: "Nove",
                none: "Nenhume",
                found: "Encontrade",
                all: "Todes",
                active: "Ative",
                inactive: "Inative",
                archived: "Arquivade",
            };
        }
    };

    // Create config with fallbacks for all label properties
    const configWithFallbacks = {
        ...config,
        labels: {
            ...Object.keys(defaultLabels).reduce((acc, key) => {
                const value = config?.labels?.[key] || defaultLabels[key];
                acc[key] = value;
                return acc;
            }, {}),
        },
    };

    const value = {
        config: configWithFallbacks || defaultSettings,
        themeMode,
        layoutMode,
        setThemeMode,
        setLayoutMode,
        resolvedMode,
        fontSettings,
        setFonts,
        accessibility,
        setAccessibility,
        loading,
        canEditTheme,
        themePermissions,
        themeSaving,
        userThemeSettings,
        fetchUserTheme,
        saveUserTheme,
        saveThemeSettings,
        saveGlobalTheme,
        saveThemePermissions,
        allowedThemes,
        globalTheme,
        updateSettings,
        getPersistentFilter,
        setPersistentFilter,
        getGenderHelpers,
    };

    return (
        <ConfigContext.Provider value={value}>
            {children}
        </ConfigContext.Provider>
    );
};

export default ConfigProvider;
