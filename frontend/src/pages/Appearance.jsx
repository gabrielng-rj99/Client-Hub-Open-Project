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

import React, { useState, useEffect, useCallback, useRef } from "react";
import { useConfig, THEME_PRESETS } from "../contexts/ConfigContext";
import { themeApi } from "../api/themeApi";
import "./styles/Appearance.css";

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

export default function Appearance({ token, apiUrl }) {
    const config = useConfig();
    const canEditTheme = config?.canEditTheme || false;
    const themePermissions = config?.themePermissions || {};
    const themeSaving = config?.themeSaving || false;
    const saveThemeSettings = config?.saveThemeSettings || (() => { });
    const themeMode = config?.themeMode || "system";
    const setThemeMode = config?.setThemeMode || (() => { });
    const layoutMode = config?.layoutMode || "standard";
    const setLayoutMode = config?.setLayoutMode || (() => { });
    const accessibility = config?.accessibility || {};
    const setAccessibility = config?.setAccessibility || (() => { });
    const userThemeSettings = config?.userThemeSettings || null;
    const fetchUserTheme = config?.fetchUserTheme || (() => { });
    const fontSettings = config?.fontSettings || {
        general: "System",
        title: "System",
        tableTitle: "System",
        tableContent: "System",
    };
    const setFonts = config?.setFonts || (() => { });
    const saveGlobalTheme = config?.saveGlobalTheme || (() => { });
    const saveThemePermissions = config?.saveThemePermissions || (() => { });
    // Consolidated data from context (avoids separate API calls)
    const contextAllowedThemes = config?.allowedThemes || [];
    const contextGlobalTheme = config?.globalTheme || null;

    // Role permissions state for theme sync
    const [roles, setRoles] = useState([]);
    const [rolePermissions, setRolePermissions] = useState({});
    const [themeUpdatePermissionId, setThemeUpdatePermissionId] =
        useState(null);
    const [permissionStatus, setPermissionStatus] = useState("loading"); // "allow" | "block" | "custom" | "loading"
    const [showCustomWarning, setShowCustomWarning] = useState(false);
    const [pendingPermissionChange, setPendingPermissionChange] =
        useState(null);

    // Debounce timeout refs (separate refs to avoid conflicts)
    const debounceTimeout = useRef(null);
    const permissionsDebounceTimeout = useRef(null);
    // Track if user has actually interacted (not just data sync on mount)
    const hasUserInteracted = useRef(false);

    // Get the best available theme data - prioritize userThemeSettings from API
    const getInitialTheme = () => {
        if (userThemeSettings && userThemeSettings.preset) {
            return userThemeSettings;
        }
        return config.config?.theme || config.theme || {};
    };

    // State for saved theme to identify current saved theme
    const [savedTheme, setSavedTheme] = useState(getInitialTheme);

    // Form data for theme selection
    const [formData, setFormData] = useState({
        theme: getInitialTheme(),
    });

    // Global theme (default for new users)
    const [globalTheme, setGlobalTheme] = useState(null);

    // Form data for permissions
    const [permissionsFormData, setPermissionsFormData] = useState({
        usersCanEditTheme: themePermissions?.usersCanEditTheme || false,
        adminsCanEditTheme: themePermissions?.adminsCanEditTheme || true,
    });

    // Initialize with all valid themes from THEME_PRESETS (all allowed by default)
    // Using useMemo to prevent recreating the array on every render
    const allValidThemes = React.useMemo(
        () => Object.keys(THEME_PRESETS).concat(["custom"]),
        [],
    );
    const [allowedThemes, setAllowedThemes] = useState(() =>
        Object.keys(THEME_PRESETS).concat(["custom"]),
    );
    const [blockedThemes, setBlockedThemes] = useState([]);

    // Multi-select and drag-drop state
    const [selectedThemes, setSelectedThemes] = useState(new Set());
    const [selectionStart, setSelectionStart] = useState(null);
    const [draggedThemes, setDraggedThemes] = useState(null);
    const [isDragging, setIsDragging] = useState(false);
    const [isLassoSelecting, setIsLassoSelecting] = useState(false);
    const [lassoStart, setLassoStart] = useState(null);
    const [lassoRect, setLassoRect] = useState(null);
    const [currentZone, setCurrentZone] = useState(null);

    // Get user role from localStorage
    const userRole = localStorage.getItem("userRole") || "user";
    const isRoot = userRole === "root";

    // Active section tab
    const [activeTab, setActiveTab] = useState("general");

    // Error state for validation messages
    const [error, setError] = useState(null);

    // NOTE: fetchUserTheme is NOT called here — ConfigContext already loads
    // user theme on mount. Re-calling it would cause a duplicate /api/user/theme request.

    // Load roles and their permissions lazily when the themes tab is active
    const [rolesLoaded, setRolesLoaded] = useState(false);
    useEffect(() => {
        if (isRoot && activeTab === "themes" && !rolesLoaded) {
            loadRolesAndPermissions();
            setRolesLoaded(true);
        }
    }, [isRoot, activeTab, rolesLoaded]);

    const loadRolesAndPermissions = async () => {
        try {
            const authToken = localStorage.getItem("accessToken");
            if (!authToken) return;

            // Load roles
            const rolesResponse = await fetch("/api/roles", {
                headers: { Authorization: `Bearer ${authToken}` },
            });
            if (!rolesResponse.ok) return;
            const rolesData = await rolesResponse.json();
            const loadedRoles = rolesData.data || [];
            setRoles(loadedRoles);

            // Load all permissions to find theme:update
            const permsResponse = await fetch("/api/permissions", {
                headers: { Authorization: `Bearer ${authToken}` },
            });
            if (!permsResponse.ok) return;
            const permsData = await permsResponse.json();
            const allPerms = permsData.data || [];

            // Find theme:update permission ID
            const themeUpdatePerm = allPerms.find(
                (p) => p.resource === "theme" && p.action === "update",
            );
            if (themeUpdatePerm) {
                setThemeUpdatePermissionId(themeUpdatePerm.id);
            }

            // Load permissions for each non-root role
            const permMap = {};
            for (const role of loadedRoles) {
                if (role.name === "root") continue;
                try {
                    const rolePermsResponse = await fetch(
                        `/api/roles/${role.id}/permissions`,
                        { headers: { Authorization: `Bearer ${authToken}` } },
                    );
                    if (rolePermsResponse.ok) {
                        const rolePermsData = await rolePermsResponse.json();
                        permMap[role.id] = rolePermsData.data || [];
                    }
                } catch (e) {
                    console.error(
                        `Error loading permissions for role ${role.name}:`,
                        e,
                    );
                }
            }
            setRolePermissions(permMap);
        } catch (err) {
            console.error("Error loading roles and permissions:", err);
        }
    };

    // Determine permission status based on role permissions
    useEffect(() => {
        if (!themeUpdatePermissionId || roles.length === 0) {
            return;
        }

        const nonRootRoles = roles.filter((r) => r.name !== "root");
        if (nonRootRoles.length === 0) {
            setPermissionStatus("allow");
            return;
        }

        let allHavePermission = true;
        let noneHavePermission = true;

        for (const role of nonRootRoles) {
            const perms = rolePermissions[role.id] || [];
            const hasThemeUpdate = perms.some(
                (p) => p.id === themeUpdatePermissionId,
            );

            if (hasThemeUpdate) {
                noneHavePermission = false;
            } else {
                allHavePermission = false;
            }
        }

        if (allHavePermission) {
            setPermissionStatus("allow");
        } else if (noneHavePermission) {
            setPermissionStatus("block");
        } else {
            setPermissionStatus("custom");
        }
    }, [roles, rolePermissions, themeUpdatePermissionId]);

    // Sync form data and saved theme when userThemeSettings or config loads
    useEffect(() => {
        // Prioritize userThemeSettings (from API) over config.config.theme
        let themeData;
        if (userThemeSettings && userThemeSettings.preset) {
            themeData = userThemeSettings;
        } else {
            themeData = config.config?.theme || config.theme || {};
        }

        setFormData({
            theme: themeData,
        });
        setSavedTheme(themeData);
    }, [userThemeSettings, config.config?.theme, config.theme]);

    // Sync global theme from context (loaded by ConfigContext in single API call)
    useEffect(() => {
        if (isRoot && contextGlobalTheme) {
            const globalPreset =
                contextGlobalTheme.preset ||
                contextGlobalTheme.theme_preset ||
                "default";
            setGlobalTheme(globalPreset);
        }
    }, [isRoot, contextGlobalTheme]);

    // Sync permissions form data
    useEffect(() => {
        setPermissionsFormData({
            usersCanEditTheme: themePermissions?.usersCanEditTheme || false,
            adminsCanEditTheme: themePermissions?.adminsCanEditTheme || true,
        });
    }, [themePermissions]);

    // Sync allowed themes from context (loaded by ConfigContext in single API call)
    useEffect(() => {
        if (contextAllowedThemes && contextAllowedThemes.length > 0) {
            // Filter to only valid themes from THEME_PRESETS
            const validAllowedThemes = contextAllowedThemes.filter((t) =>
                allValidThemes.includes(t),
            );

            // If filtering removed some themes, use only valid ones
            if (validAllowedThemes.length > 0) {
                setAllowedThemes(validAllowedThemes);
                setBlockedThemes(
                    allValidThemes.filter(
                        (t) => !validAllowedThemes.includes(t),
                    ),
                );
            }
        }
    }, [contextAllowedThemes, allValidThemes]);

    // Apply theme preview when user selects a theme
    useEffect(() => {
        applyThemePreview(formData.theme);
    }, [formData.theme]);

    const applyThemePreview = (themeData) => {
        const root = document.documentElement;
        let themeColors;

        if (themeData.preset === "custom") {
            themeColors = {
                buttonPrimary: themeData.primaryColor || "#0284c7",
                buttonSecondary: themeData.secondaryColor || "#0369a1",
                bgPage: themeData.backgroundColor || "#f0f9ff",
                bgCard: themeData.surfaceColor || "#ffffff",
                textPrimary: themeData.textColor || "#0c4a6e",
                textSecondary: themeData.textSecondaryColor || "#475569",
                borderDefault: themeData.borderColor || "#7dd3fc",
            };
        } else if (THEME_PRESETS[themeData.preset]) {
            const currentThemeMode =
                localStorage.getItem("themeMode") || "system";
            const prefersDark = window.matchMedia(
                "(prefers-color-scheme: dark)",
            ).matches;
            const isDark =
                currentThemeMode === "dark" ||
                (currentThemeMode === "system" && prefersDark);

            themeColors = isDark
                ? THEME_PRESETS[themeData.preset].dark
                : THEME_PRESETS[themeData.preset].light;
        } else {
            return;
        }

        // Use text colors directly from theme
        const primaryTextColor = themeColors.textPrimary || "#222222";
        const secondaryTextColor = themeColors.textSecondary || "#444444";

        // Apply CSS variables for instant preview
        root.style.setProperty("--primary-color", themeColors.buttonPrimary);
        root.style.setProperty(
            "--secondary-color",
            themeColors.buttonSecondary,
        );
        root.style.setProperty("--bg-color", themeColors.bgPage);
        root.style.setProperty("--content-bg", themeColors.bgCard);
        root.style.setProperty("--border-color", themeColors.borderDefault);
        root.style.setProperty("--primary-text-color", primaryTextColor);
        root.style.setProperty("--secondary-text-color", secondaryTextColor);
        root.style.setProperty(
            "--nav-text-color",
            themeColors.textNav || secondaryTextColor,
        );
        root.style.setProperty(
            "--app-bg-color",
            themeColors.bgNavbar || themeColors.buttonSecondary,
        );
        root.style.setProperty(
            "--page-title-color",
            themeColors.textTitle || themeColors.textPrimary,
        );
        root.style.setProperty(
            "--active-nav-text-color",
            themeColors.textNavActive || primaryTextColor,
        );
        root.style.setProperty(
            "--primary-dark",
            darkenColor(themeColors.buttonPrimary, 10),
        );
        root.style.setProperty(
            "--secondary-light",
            darkenColor(themeColors.buttonSecondary, 10),
        );
    };

    const saveThemeImmediate = useCallback(
        async (themeData) => {
            try {
                await saveThemeSettings(themeData);
                setSavedTheme(themeData); // Update saved theme
            } catch (err) {
                console.error("Error saving theme:", err);
            }
        },
        [saveThemeSettings],
    );

    const debouncedSaveTheme = useCallback(
        (themeData) => {
            // Only save if user has actually interacted (prevents save on mount sync)
            if (!hasUserInteracted.current) {
                return;
            }
            if (debounceTimeout.current) clearTimeout(debounceTimeout.current);
            debounceTimeout.current = setTimeout(() => {
                saveThemeSettings(themeData);
                setSavedTheme(themeData); // Update saved theme
            }, 500);
        },
        [saveThemeSettings],
    );

    const handleThemeChange = async (key, value) => {
        // Validate that the preset is allowed (if changing preset)
        if (key === "preset" && !allowedThemes.includes(value)) {
            setError(`O tema "${value}" não está disponível no momento.`);
            return;
        }

        // Build the new theme data
        let newThemeData = {
            ...formData.theme,
        };

        if (key === "preset") {
            // When changing preset, reset all custom colors to null
            // This ensures the preset's colors take precedence and we save a clean state
            newThemeData = {
                ...newThemeData,
                preset: value,
                primaryColor: null,
                secondaryColor: null,
                backgroundColor: null,
                surfaceColor: null,
                textColor: null,
                textSecondaryColor: null,
                borderColor: null,
            };
        } else {
            newThemeData[key] = value;
        }

        // Update local state
        setFormData((prev) => ({
            ...prev,
            theme: newThemeData,
        }));

        // Apply theme to UI immediately
        applyThemePreview(newThemeData);

        // Save directly to database
        try {
            await saveThemeImmediate(newThemeData);
        } catch (err) {
            setError(`Erro ao salvar tema: ${err.message}`);
        }
    };

    const debouncedSavePermissions = useCallback(() => {
        // Only save if user has actually interacted
        if (!hasUserInteracted.current) {
            return;
        }
        if (permissionsDebounceTimeout.current)
            clearTimeout(permissionsDebounceTimeout.current);
        permissionsDebounceTimeout.current = setTimeout(() => {
            if (!isRoot) return;
            saveThemePermissions(
                permissionsFormData.usersCanEditTheme,
                permissionsFormData.adminsCanEditTheme,
            ).catch((err) => {
                console.error("Error saving permissions:", err);
            });
        }, 500);
    }, [permissionsFormData, isRoot, saveThemePermissions]);

    const handlePermissionsChange = (key, value) => {
        // Mark that user has interacted
        hasUserInteracted.current = true;
        setPermissionsFormData((prev) => ({
            ...prev,
            [key]: value,
        }));
        debouncedSavePermissions();
    };

    // Handle permission status button click (Allow/Block/Custom)
    const handlePermissionStatusChange = async (newStatus) => {
        if (newStatus === permissionStatus) return;

        // If current status is custom and user wants to change, show warning
        if (permissionStatus === "custom" && newStatus !== "custom") {
            setPendingPermissionChange(newStatus);
            setShowCustomWarning(true);
            return;
        }

        await applyPermissionStatus(newStatus);
    };

    const applyPermissionStatus = async (newStatus) => {
        if (!themeUpdatePermissionId) return;

        const authToken = localStorage.getItem("accessToken");
        if (!authToken) return;

        try {
            const nonRootRoles = roles.filter((r) => r.name !== "root");

            for (const role of nonRootRoles) {
                const currentPerms = rolePermissions[role.id] || [];
                let newPermIds;

                if (newStatus === "allow") {
                    // Add theme:update permission if not present
                    const hasIt = currentPerms.some(
                        (p) => p.id === themeUpdatePermissionId,
                    );
                    if (!hasIt) {
                        newPermIds = [
                            ...currentPerms.map((p) => p.id),
                            themeUpdatePermissionId,
                        ];
                    } else {
                        continue;
                    }
                } else if (newStatus === "block") {
                    // Remove theme:update permission
                    newPermIds = currentPerms
                        .filter((p) => p.id !== themeUpdatePermissionId)
                        .map((p) => p.id);
                } else {
                    continue;
                }

                await fetch(`/api/roles/${role.id}/permissions`, {
                    method: "PUT",
                    headers: {
                        "Content-Type": "application/json",
                        Authorization: `Bearer ${authToken}`,
                    },
                    body: JSON.stringify({ permission_ids: newPermIds }),
                });
            }

            // Also update system settings
            await saveThemePermissions(
                newStatus === "allow",
                newStatus === "allow",
            );

            // Reload permissions
            await loadRolesAndPermissions();
            setPermissionStatus(newStatus);
        } catch (err) {
            console.error("Error applying permission status:", err);
        }
    };

    const confirmCustomWarning = async () => {
        setShowCustomWarning(false);
        if (pendingPermissionChange) {
            await applyPermissionStatus(pendingPermissionChange);
            setPendingPermissionChange(null);
        }
    };

    const cancelCustomWarning = () => {
        setShowCustomWarning(false);
        setPendingPermissionChange(null);
    };

    const handleSaveGlobalTheme = useCallback(async () => {
        if (!isRoot) {
            console.error("Only root can set global theme");
            return;
        }

        // Use preview theme (formData) which reflects what user is currently seeing
        const themeToSave =
            formData.theme ||
            config.config?.theme ||
            config.theme ||
            savedTheme ||
            {};
        const preset = themeToSave.preset || "default";

        try {
            // First save the user's current theme to ensure it's persisted
            if (formData.theme && formData.theme.preset) {
                await saveThemeImmediate(formData.theme);
            }

            // Then save as global theme
            const response = await fetch(`${apiUrl}/settings/global-theme`, {
                method: "PUT",
                headers: {
                    "Content-Type": "application/json",
                    Authorization: `Bearer ${token}`,
                },
                body: JSON.stringify({
                    theme_preset: preset,
                    primary_color: themeToSave.primaryColor || null,
                    secondary_color: themeToSave.secondaryColor || null,
                    background_color: themeToSave.backgroundColor || null,
                    surface_color: themeToSave.surfaceColor || null,
                    text_color: themeToSave.textColor || null,
                    text_secondary_color:
                        themeToSave.textSecondaryColor || null,
                    border_color: themeToSave.borderColor || null,
                }),
            });

            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(
                    errorData.error || "Erro ao salvar tema global",
                );
            }

            // Update global theme state
            setGlobalTheme(preset);

            // Get theme display name
            const themeName =
                preset === "custom"
                    ? "Personalizado"
                    : THEME_PRESETS[preset]?.name || preset;
            alert(
                `Tema "${themeName}" definido como padrão para novos usuários!`,
            );
        } catch (err) {
            console.error("❌ Error saving global theme:", err);
            alert(`Erro ao salvar tema global: ${err.message}`);
        }
    }, [
        formData,
        config,
        savedTheme,
        token,
        isRoot,
        apiUrl,
        saveThemeImmediate,
    ]);

    const moveThemeToBlocked = (theme) => {
        if (blockedThemes.includes(theme)) return;
        setAllowedThemes((prev) => prev.filter((t) => t !== theme));
        setBlockedThemes((prev) => [...prev, theme]);
    };

    const moveThemeToAllowed = (theme) => {
        if (allowedThemes.includes(theme)) return;
        setBlockedThemes((prev) => prev.filter((t) => t !== theme));
        setAllowedThemes((prev) => [...prev, theme]);
    };

    const debouncedSaveAllowedThemes = useCallback(async () => {
        if (debounceTimeout.current) clearTimeout(debounceTimeout.current);
        debounceTimeout.current = setTimeout(async () => {
            if (!isRoot) return;
            try {
                const validThemesToSave = allowedThemes.filter((t) =>
                    allValidThemes.includes(t),
                );
                if (validThemesToSave.length === 0) return;
                const response = await fetch(
                    `${apiUrl}/settings/allowed-themes`,
                    {
                        method: "PUT",
                        headers: {
                            "Content-Type": "application/json",
                            Authorization: `Bearer ${token}`,
                        },
                        body: JSON.stringify({
                            allowed_themes: validThemesToSave,
                        }),
                    },
                );
                if (!response.ok) {
                    throw new Error("Erro ao salvar temas permitidos");
                }
            } catch (err) {
                console.error("Error saving allowed themes:", err);
            }
        }, 1000);
    }, [allowedThemes, allValidThemes, token, isRoot, apiUrl]);

    const saveAllowedThemesImmediate = useCallback(
        async (themesToSave) => {
            if (!isRoot) return;
            try {
                const validThemesToSave = themesToSave.filter((t) =>
                    allValidThemes.includes(t),
                );
                if (validThemesToSave.length === 0) return;
                const response = await fetch(
                    `${apiUrl}/settings/allowed-themes`,
                    {
                        method: "PUT",
                        headers: {
                            "Content-Type": "application/json",
                            Authorization: `Bearer ${token}`,
                        },
                        body: JSON.stringify({
                            allowed_themes: validThemesToSave,
                        }),
                    },
                );
                if (!response.ok) {
                    throw new Error("Erro ao salvar temas permitidos");
                }
            } catch (err) {
                console.error("Error saving allowed themes:", err);
            }
        },
        [allValidThemes, token, isRoot, apiUrl],
    );

    const handleBlockAllExceptSelected = async () => {
        // Get the current theme preset being used/previewed
        const currentPreset =
            formData.theme?.preset ||
            config.config?.theme?.preset ||
            config.theme?.preset ||
            savedTheme?.preset ||
            "default";

        // Get all themes
        const allThemes = Object.keys(THEME_PRESETS).concat(["custom"]);

        // Build list of themes to keep allowed (current + global if different)
        const themesToKeep = new Set([currentPreset]);

        // Also keep the global theme (default for new users) if it exists and is different
        if (globalTheme && globalTheme !== currentPreset) {
            themesToKeep.add(globalTheme);
        }

        const newAllowedThemes = Array.from(themesToKeep);
        const newBlockedThemes = allThemes.filter((t) => !themesToKeep.has(t));

        // Update state
        setAllowedThemes(newAllowedThemes);
        setBlockedThemes(newBlockedThemes);

        // Save to backend with the new values directly
        await saveAllowedThemesImmediate(newAllowedThemes);
    };

    const handleUnlockAllThemes = async () => {
        const allThemes = Object.keys(THEME_PRESETS).concat(["custom"]);
        setAllowedThemes(allThemes);
        setBlockedThemes([]);
        await saveAllowedThemesImmediate(allThemes);
    };

    // Multi-select functions
    const toggleThemeSelection = (theme) => {
        setSelectedThemes((prev) => {
            const newSet = new Set(prev);
            if (newSet.has(theme)) {
                newSet.delete(theme);
            } else {
                newSet.add(theme);
            }
            return newSet;
        });
    };

    const selectThemeRange = (endTheme, list) => {
        if (!selectionStart) return;

        const startIdx = list.indexOf(selectionStart);
        const endIdx = list.indexOf(endTheme);

        if (startIdx === -1 || endIdx === -1) return;

        const [minIdx, maxIdx] =
            startIdx < endIdx ? [startIdx, endIdx] : [endIdx, startIdx];

        const newSet = new Set(selectedThemes);
        for (let i = minIdx; i <= maxIdx; i++) {
            newSet.add(list[i]);
        }
        setSelectedThemes(newSet);
    };

    // Drag and drop functions
    const handleThemeDragStart = (theme, event) => {
        let themesToDrag = [theme];

        if (selectedThemes.has(theme)) {
            themesToDrag = Array.from(selectedThemes);
        } else {
            setSelectedThemes(new Set([theme]));
            themesToDrag = [theme];
        }

        setDraggedThemes(themesToDrag);
        setIsDragging(true);
        event.dataTransfer.effectAllowed = "move";
        event.dataTransfer.setData("themes", JSON.stringify(themesToDrag));
    };

    const handleContainerDragOver = (event) => {
        event.preventDefault();
        event.dataTransfer.dropEffect = "move";
        event.currentTarget.classList.add("drag-over");
    };

    const handleContainerDragLeave = (event) => {
        if (event.currentTarget === event.target) {
            event.currentTarget.classList.remove("drag-over");
        }
    };

    const handleDropToBlocked = (event) => {
        event.preventDefault();
        event.currentTarget.classList.remove("drag-over");
        if (!draggedThemes) return;

        const currentPreset = savedTheme.preset;
        let themesToMove = draggedThemes.filter(
            (t) => !blockedThemes.includes(t),
        );

        if (themesToMove.length === 0) return;

        // Prevent root from blocking their current theme
        if (isRoot && themesToMove.includes(currentPreset)) {
            themesToMove = themesToMove.filter((t) => t !== currentPreset);
            if (themesToMove.length === 0) return;
        }

        // Calculate new allowed themes (maintaining order of remaining themes)
        const newAllowed = allowedThemes.filter(
            (t) => !themesToMove.includes(t),
        );

        setAllowedThemes(newAllowed);
        setBlockedThemes((prev) => [...prev, ...themesToMove]);

        // If current theme is being blocked and not root, switch to default
        if (themesToMove.includes(currentPreset)) {
            if (isRoot) {
                // Root can't block their theme, so no change
            } else {
                handleThemeChange("preset", "default");
            }
        }

        setSelectedThemes(new Set());
        setDraggedThemes(null);
        setIsDragging(false);
        saveAllowedThemesImmediate(newAllowed);
    };

    const handleDropToAllowed = (event) => {
        event.preventDefault();
        event.currentTarget.classList.remove("drag-over");
        if (!draggedThemes) return;

        const themesToMove = draggedThemes.filter(
            (t) => !allowedThemes.includes(t),
        );

        if (themesToMove.length === 0) return;

        // Maintain order: remove from blocked and insert at beginning to preserve sequence
        const newBlocked = blockedThemes.filter(
            (t) => !themesToMove.includes(t),
        );

        setBlockedThemes(newBlocked);
        setAllowedThemes((prev) => [...themesToMove, ...prev]);

        setSelectedThemes(new Set());
        setDraggedThemes(null);
        setIsDragging(false);
        saveAllowedThemesImmediate([
            ...themesToMove,
            ...allowedThemes.filter((t) => !themesToMove.includes(t)),
        ]);
    };

    // Lasso selection handlers
    const handleLassoStart = (event, zone) => {
        // Only start if clicking on empty space
        if (event.target.closest(".preset-card") || event.button !== 0) {
            return;
        }

        const container = event.currentTarget;
        const rect = container.getBoundingClientRect();
        const startX = event.clientX - rect.left;
        const startY = event.clientY - rect.top;

        setCurrentZone(zone);
        setLassoStart({ x: startX, y: startY });
        setIsLassoSelecting(true);
    };

    const handleLassoMove = (event) => {
        if (!isLassoSelecting || !lassoStart || !currentZone) return;

        const container = event.currentTarget;
        const rect = container.getBoundingClientRect();
        const currentX = event.clientX - rect.left;
        const currentY = event.clientY - rect.top;

        const newRect = {
            left: Math.min(lassoStart.x, currentX),
            top: Math.min(lassoStart.y, currentY),
            right: Math.max(lassoStart.x, currentX),
            bottom: Math.max(lassoStart.y, currentY),
        };

        setLassoRect(newRect);

        // Select items in rect
        const cards = container.querySelectorAll(".preset-card");
        const selected = new Set();

        cards.forEach((card) => {
            const cardRect = card.getBoundingClientRect();
            const cardLeft = cardRect.left - rect.left;
            const cardTop = cardRect.top - rect.top;
            const cardRight = cardLeft + cardRect.width;
            const cardBottom = cardTop + cardRect.height;

            // Check if card intersects with lasso rect
            if (
                cardLeft < newRect.right &&
                cardRight > newRect.left &&
                cardTop < newRect.bottom &&
                cardBottom > newRect.top
            ) {
                const theme = card.getAttribute("data-theme");
                if (theme) {
                    selected.add(theme);
                }
            }
        });

        setSelectedThemes(selected);
    };

    const handleLassoEnd = () => {
        setIsLassoSelecting(false);
        setLassoStart(null);
        setLassoRect(null);
        setCurrentZone(null);
    };

    // handleSaveAllowedThemes is now debounced via debouncedSaveAllowedThemes

    const handleThemeModeChange = (mode) => {
        setThemeMode(mode);
        localStorage.setItem("themeMode", mode);
    };

    const handleAccessibilityChange = (key, value) => {
        setAccessibility({ [key]: value });
    };

    const handleFontChange = (key, value) => {
        setFonts({ [key]: value });
    };

    const saveAccessibility = async (newAccessibility) => {
        setAccessibility(newAccessibility);
    };

    const getThemeName = (themeKey) => {
        if (themeKey === "custom") return "Personalizado";
        return THEME_PRESETS[themeKey]?.name || themeKey;
    };

    return (
        <div className="appearance-container">
            <div className="appearance-header">
                <h1 className="appearance-title">🎨 Aparência</h1>
            </div>

            {/* Error Message */}
            {error && (
                <div
                    className="error-message"
                    style={{
                        padding: "12px 16px",
                        marginBottom: "16px",
                        backgroundColor: "#fee2e2",
                        border: "1px solid #fca5a5",
                        borderRadius: "8px",
                        color: "#dc2626",
                        display: "flex",
                        justifyContent: "space-between",
                        alignItems: "center",
                    }}
                >
                    <span>⚠️ {error}</span>
                    <button
                        onClick={() => setError(null)}
                        style={{
                            background: "none",
                            border: "none",
                            cursor: "pointer",
                            fontSize: "18px",
                            color: "#dc2626",
                        }}
                    >
                        ×
                    </button>
                </div>
            )}

            {/* Main Navigation Tabs */}
            <div className="settings-main-tabs">
                <button
                    className={`settings-main-tab ${activeTab === "general" ? "active" : ""}`}
                    onClick={() => setActiveTab("general")}
                >
                    ⚙️ Geral
                </button>
                <button
                    className={`settings-main-tab ${activeTab === "themes" ? "active" : ""}`}
                    onClick={() => setActiveTab("themes")}
                >
                    🎨 Temas e Cores
                </button>
            </div>

            {/* TAB: GERAL */}
            {activeTab === "general" && (
                <section className="appearance-section appearance-section-nav-color">
                    <div className="appearance-grid-layout">
                        {/* 1. Display Mode */}
                        <div className="grid-item">
                            <h2>🖥️ Modo de Exibição</h2>
                            <div className="display-mode-container">
                                <div className="theme-mode-selector">
                                    <button
                                        className={`theme-mode-button ${themeMode === "light" ? "active" : ""}`}
                                        onClick={() =>
                                            handleThemeModeChange("light")
                                        }
                                    >
                                        <span className="mode-icon">☀️</span>
                                        <span className="mode-label">
                                            Claro
                                        </span>
                                    </button>
                                    <button
                                        className={`theme-mode-button ${themeMode === "dark" ? "active" : ""}`}
                                        onClick={() =>
                                            handleThemeModeChange("dark")
                                        }
                                    >
                                        <span className="mode-icon">🌙</span>
                                        <span className="mode-label">
                                            Escuro
                                        </span>
                                    </button>
                                    <button
                                        className={`theme-mode-button ${themeMode === "system" ? "active" : ""}`}
                                        onClick={() =>
                                            handleThemeModeChange("system")
                                        }
                                    >
                                        <span className="mode-icon">💻</span>
                                        <span className="mode-label">
                                            Sistema
                                        </span>
                                    </button>
                                </div>
                            </div>
                        </div>

                        {/* 2. Layout */}
                        <div className="grid-item">
                            <h2>📏 Layout</h2>
                            <div className="display-mode-container">
                                <button
                                    className={`theme-mode-button ${layoutMode === "centralized" ? "active" : ""}`}
                                    onClick={() => setLayoutMode("centralized")}
                                >
                                    <span className="mode-icon">📄</span>
                                    <span className="mode-label">
                                        Centralizado
                                    </span>
                                </button>
                                <button
                                    className={`theme-mode-button ${layoutMode === "standard" ? "active" : ""}`}
                                    onClick={() => setLayoutMode("standard")}
                                >
                                    <span className="mode-icon">🖥️</span>
                                    <span className="mode-label">Padrão</span>
                                </button>
                                <button
                                    className={`theme-mode-button ${layoutMode === "full" ? "active" : ""}`}
                                    onClick={() => setLayoutMode("full")}
                                >
                                    <span className="mode-icon">↔️</span>
                                    <span className="mode-label">
                                        Tela Cheia
                                    </span>
                                </button>
                            </div>
                        </div>

                        {/* 3. Typography */}
                        <div className="grid-item">
                            <h2>🔤 Tipografia</h2>
                            <div className="typography-grid">
                                {[
                                    { label: "Fonte Geral", key: "general" },
                                    { label: "Fonte de Títulos", key: "title" },
                                    {
                                        label: "Fonte de Títulos Secundários",
                                        key: "tableTitle",
                                    },
                                ].map(({ label, key }) => {
                                    // Standardize the list of options
                                    const standardOptions = [
                                        "System",
                                        // Linux / FLOSS Standard
                                        "DejaVu Sans",
                                        "Liberation Sans",
                                        "Ubuntu",
                                        "DejaVu Serif",
                                        "Liberation Serif",
                                        "DejaVu Sans Mono",
                                        "Liberation Mono",
                                        // Accessibility
                                        "OpenDyslexic",
                                        "OpenDyslexic 3",
                                        "OpenDyslexic Mono",
                                        // Web Fonts (Open Source)
                                        "Inter",
                                        "Roboto",
                                        "Open Sans",
                                        "Lato",
                                        "Montserrat",
                                        "Poppins",
                                        "Source Sans Pro",
                                        "Oswald",
                                        "Raleway",
                                        "Merriweather",
                                        "Nunito",
                                        "Fira Code",
                                        "Atkinson Hyperlegible",
                                        "Lexend",
                                    ];

                                    const isCustom = !standardOptions.includes(
                                        fontSettings[key],
                                    );
                                    const selectValue = isCustom
                                        ? "custom"
                                        : fontSettings[key];

                                    return (
                                        <div className="form-group" key={key}>
                                            <div
                                                style={{
                                                    marginBottom: "0.5rem",
                                                }}
                                            >
                                                <label
                                                    style={{
                                                        display: "block",
                                                        marginBottom: "0.5rem",
                                                        fontSize: "1rem",
                                                        fontWeight: "600",
                                                    }}
                                                >
                                                    {label}
                                                </label>
                                                <div
                                                    style={{
                                                        display: "flex",
                                                        gap: "10px",
                                                        alignItems: "center",
                                                    }}
                                                >
                                                    <div
                                                        className="font-controls"
                                                        style={{
                                                            display: "flex",
                                                            gap: "8px",
                                                        }}
                                                    >
                                                        <button
                                                            type="button"
                                                            className="icon-button"
                                                            title="Definir como padrão para todos os usuários"
                                                            onClick={async () => {
                                                                if (
                                                                    !window.confirm(
                                                                        "Definir a configuração de fontes atual como padrão global para todos os usuários?",
                                                                    )
                                                                )
                                                                    return;
                                                                try {
                                                                    await saveGlobalTheme(
                                                                        {
                                                                            ...formData.theme,
                                                                            fontGeneral:
                                                                                fontSettings.general,
                                                                            fontTitle:
                                                                                fontSettings.title,
                                                                            fontTableTitle:
                                                                                fontSettings.tableTitle,
                                                                        },
                                                                    );
                                                                    alert(
                                                                        "Padrão global atualizado com sucesso!",
                                                                    );
                                                                } catch (e) {
                                                                    alert(
                                                                        "Erro ao salvar: " +
                                                                        e.message,
                                                                    );
                                                                }
                                                            }}
                                                            style={{
                                                                background:
                                                                    "none",
                                                                border: "none",
                                                                cursor: "pointer",
                                                                opacity: 0.8,
                                                                fontSize:
                                                                    "1.5rem",
                                                                padding: "4px",
                                                            }}
                                                        >
                                                            💾
                                                        </button>
                                                        <button
                                                            type="button"
                                                            className="icon-button"
                                                            title={
                                                                themePermissions.usersCanEditTheme
                                                                    ? "Permitir que usuários editem (Atualmente: Permitido)"
                                                                    : "Bloquear edição por usuários (Atualmente: Bloqueado)"
                                                            }
                                                            onClick={async () => {
                                                                try {
                                                                    const newStatus =
                                                                        !themePermissions.usersCanEditTheme;
                                                                    await saveThemePermissions(
                                                                        newStatus,
                                                                        themePermissions.adminsCanEditTheme,
                                                                    );
                                                                    alert(
                                                                        newStatus
                                                                            ? "Edição por usuários desbloqueada."
                                                                            : "Edição por usuários bloqueada.",
                                                                    );
                                                                } catch (e) {
                                                                    alert(
                                                                        "Erro ao atualizar permissões: " +
                                                                        e.message,
                                                                    );
                                                                }
                                                            }}
                                                            style={{
                                                                background:
                                                                    "none",
                                                                border: "none",
                                                                cursor: "pointer",
                                                                opacity: 0.8,
                                                                fontSize:
                                                                    "1.5rem",
                                                                padding: "4px",
                                                            }}
                                                        >
                                                            {themePermissions.usersCanEditTheme
                                                                ? "🔓"
                                                                : "🔒"}
                                                        </button>
                                                    </div>
                                                    <div style={{ flex: 1 }}>
                                                        <select
                                                            value={selectValue}
                                                            onChange={(e) => {
                                                                if (
                                                                    e.target
                                                                        .value ===
                                                                    "custom"
                                                                ) {
                                                                    // Set to empty string to ensure input shows up immediately (since "" is not in standardOptions)
                                                                    handleFontChange(
                                                                        key,
                                                                        "",
                                                                    );
                                                                } else {
                                                                    handleFontChange(
                                                                        key,
                                                                        e.target
                                                                            .value,
                                                                    );
                                                                }
                                                            }}
                                                            className="form-control"
                                                        >
                                                            <optgroup label="Padrão">
                                                                <option
                                                                    value="System"
                                                                    style={{
                                                                        fontFamily:
                                                                            "system-ui",
                                                                    }}
                                                                >
                                                                    Sistema
                                                                    (Padrão)
                                                                </option>
                                                            </optgroup>
                                                            <optgroup label="Linux & FLOSS Standards">
                                                                <option
                                                                    value="DejaVu Sans"
                                                                    style={{
                                                                        fontFamily:
                                                                            "DejaVu Sans",
                                                                    }}
                                                                >
                                                                    DejaVu Sans
                                                                </option>
                                                                <option
                                                                    value="Liberation Sans"
                                                                    style={{
                                                                        fontFamily:
                                                                            "Liberation Sans",
                                                                    }}
                                                                >
                                                                    Liberation
                                                                    Sans
                                                                </option>
                                                                <option
                                                                    value="Ubuntu"
                                                                    style={{
                                                                        fontFamily:
                                                                            "Ubuntu",
                                                                    }}
                                                                >
                                                                    Ubuntu
                                                                </option>
                                                                <option
                                                                    value="DejaVu Serif"
                                                                    style={{
                                                                        fontFamily:
                                                                            "DejaVu Serif",
                                                                    }}
                                                                >
                                                                    DejaVu Serif
                                                                </option>
                                                                <option
                                                                    value="Liberation Serif"
                                                                    style={{
                                                                        fontFamily:
                                                                            "Liberation Serif",
                                                                    }}
                                                                >
                                                                    Liberation
                                                                    Serif
                                                                </option>
                                                                <option
                                                                    value="DejaVu Sans Mono"
                                                                    style={{
                                                                        fontFamily:
                                                                            "DejaVu Sans Mono",
                                                                    }}
                                                                >
                                                                    DejaVu Sans
                                                                    Mono
                                                                </option>
                                                                <option
                                                                    value="Liberation Mono"
                                                                    style={{
                                                                        fontFamily:
                                                                            "Liberation Mono",
                                                                    }}
                                                                >
                                                                    Liberation
                                                                    Mono
                                                                </option>
                                                            </optgroup>
                                                            <optgroup label="Acessibilidade / Dislexia">
                                                                <option
                                                                    value="OpenDyslexic"
                                                                    style={{
                                                                        fontFamily:
                                                                            "OpenDyslexic",
                                                                    }}
                                                                >
                                                                    OpenDyslexic
                                                                </option>
                                                                <option
                                                                    value="OpenDyslexic 3"
                                                                    style={{
                                                                        fontFamily:
                                                                            "OpenDyslexic3",
                                                                    }}
                                                                >
                                                                    OpenDyslexic
                                                                    3
                                                                </option>
                                                                <option
                                                                    value="OpenDyslexic Mono"
                                                                    style={{
                                                                        fontFamily:
                                                                            "OpenDyslexicMono",
                                                                    }}
                                                                >
                                                                    OpenDyslexic
                                                                    Mono
                                                                </option>
                                                            </optgroup>
                                                            <optgroup label="Web Fonts (Google Open Source)">
                                                                <option
                                                                    value="Open Sans"
                                                                    style={{
                                                                        fontFamily:
                                                                            "Open Sans",
                                                                    }}
                                                                >
                                                                    Open Sans
                                                                </option>
                                                                <option
                                                                    value="Roboto"
                                                                    style={{
                                                                        fontFamily:
                                                                            "Roboto",
                                                                    }}
                                                                >
                                                                    Roboto
                                                                </option>
                                                                <option
                                                                    value="Inter"
                                                                    style={{
                                                                        fontFamily:
                                                                            "Inter",
                                                                    }}
                                                                >
                                                                    Inter
                                                                </option>
                                                                <option
                                                                    value="Lato"
                                                                    style={{
                                                                        fontFamily:
                                                                            "Lato",
                                                                    }}
                                                                >
                                                                    Lato
                                                                </option>
                                                                <option
                                                                    value="Montserrat"
                                                                    style={{
                                                                        fontFamily:
                                                                            "Montserrat",
                                                                    }}
                                                                >
                                                                    Montserrat
                                                                </option>
                                                                <option
                                                                    value="Poppins"
                                                                    style={{
                                                                        fontFamily:
                                                                            "Poppins",
                                                                    }}
                                                                >
                                                                    Poppins
                                                                </option>
                                                                <option
                                                                    value="Fira Code"
                                                                    style={{
                                                                        fontFamily:
                                                                            "Fira Code, monospace",
                                                                    }}
                                                                >
                                                                    Fira Code
                                                                    (Monospace)
                                                                </option>
                                                                <option
                                                                    value="Source Sans Pro"
                                                                    style={{
                                                                        fontFamily:
                                                                            "Source Sans Pro",
                                                                    }}
                                                                >
                                                                    Source Sans
                                                                    Pro
                                                                </option>
                                                                <option
                                                                    value="Oswald"
                                                                    style={{
                                                                        fontFamily:
                                                                            "Oswald",
                                                                    }}
                                                                >
                                                                    Oswald
                                                                </option>
                                                                <option
                                                                    value="Raleway"
                                                                    style={{
                                                                        fontFamily:
                                                                            "Raleway",
                                                                    }}
                                                                >
                                                                    Raleway
                                                                </option>
                                                                <option
                                                                    value="Merriweather"
                                                                    style={{
                                                                        fontFamily:
                                                                            "Merriweather, serif",
                                                                    }}
                                                                >
                                                                    Merriweather
                                                                </option>
                                                                <option
                                                                    value="Nunito"
                                                                    style={{
                                                                        fontFamily:
                                                                            "Nunito",
                                                                    }}
                                                                >
                                                                    Nunito
                                                                </option>
                                                                <option
                                                                    value="Atkinson Hyperlegible"
                                                                    style={{
                                                                        fontFamily:
                                                                            "Atkinson Hyperlegible",
                                                                    }}
                                                                >
                                                                    Atkinson
                                                                    Hyperlegible
                                                                </option>
                                                                <option
                                                                    value="Lexend"
                                                                    style={{
                                                                        fontFamily:
                                                                            "Lexend",
                                                                    }}
                                                                >
                                                                    Lexend
                                                                </option>
                                                            </optgroup>
                                                            <option value="custom">
                                                                ✨
                                                                Personalizado...
                                                            </option>
                                                        </select>

                                                        {(selectValue ===
                                                            "custom" ||
                                                            isCustom) && (
                                                                <div
                                                                    style={{
                                                                        marginTop:
                                                                            "8px",
                                                                    }}
                                                                >
                                                                    <input
                                                                        type="text"
                                                                        className="form-control"
                                                                        placeholder="Nome da Google Font (ex: Ubuntu)"
                                                                        value={
                                                                            fontSettings[
                                                                                key
                                                                            ] ===
                                                                                "System"
                                                                                ? ""
                                                                                : fontSettings[
                                                                                key
                                                                                ]
                                                                        }
                                                                        onChange={(
                                                                            e,
                                                                        ) =>
                                                                            handleFontChange(
                                                                                key,
                                                                                e
                                                                                    .target
                                                                                    .value,
                                                                            )
                                                                        }
                                                                    />
                                                                    <small
                                                                        style={{
                                                                            display:
                                                                                "block",
                                                                            marginTop:
                                                                                "4px",
                                                                            color: "var(--secondary-text-color)",
                                                                        }}
                                                                    >
                                                                        ⚠️ Digite o
                                                                        nome exato
                                                                        da fonte
                                                                        disponível
                                                                        no{" "}
                                                                        <a
                                                                            href="https://fonts.google.com"
                                                                            target="_blank"
                                                                            rel="noopener noreferrer"
                                                                        >
                                                                            Google
                                                                            Fonts
                                                                        </a>
                                                                        .
                                                                    </small>
                                                                </div>
                                                            )}
                                                    </div>
                                                </div>
                                            </div>
                                        </div>
                                    );
                                })}
                            </div>
                        </div>

                        {/* 4. Accessibility */}
                        <div className="grid-item">
                            <h2>♿ Acessibilidade</h2>
                            <div className="accessibility-wrapper">
                                <div className="accessibility-section">
                                    <h3>Contraste</h3>
                                    <div className="accessibility-mode-selector">
                                        <button
                                            className={`accessibility-mode-button ${!accessibility?.highContrast ? "active" : ""}`}
                                            onClick={() =>
                                                saveAccessibility({
                                                    highContrast: false,
                                                    colorBlindMode:
                                                        accessibility.colorBlindMode,
                                                })
                                            }
                                            title="Configuração padrão"
                                        >
                                            <span className="a11y-mode-icon">
                                                👁️
                                            </span>
                                            <span className="a11y-mode-label">
                                                Padrão
                                            </span>
                                        </button>
                                        <button
                                            className={`accessibility-mode-button ${accessibility?.highContrast ? "active" : ""}`}
                                            onClick={() =>
                                                saveAccessibility({
                                                    highContrast: true,
                                                    colorBlindMode:
                                                        accessibility.colorBlindMode,
                                                })
                                            }
                                            title="Aumenta o contraste das cores escuras e claras"
                                        >
                                            <span className="a11y-mode-icon">
                                                ⚫⚪
                                            </span>
                                            <span className="a11y-mode-label">
                                                Alto Contraste
                                            </span>
                                        </button>
                                    </div>
                                </div>

                                <div className="accessibility-section">
                                    <h3>Suporte à Leitura</h3>
                                    <div className="accessibility-mode-selector">
                                        <button
                                            className={`accessibility-mode-button ${!accessibility?.dyslexicFont ? "active" : ""}`}
                                            onClick={() =>
                                                saveAccessibility({
                                                    highContrast:
                                                        accessibility.highContrast,
                                                    colorBlindMode:
                                                        accessibility.colorBlindMode,
                                                    dyslexicFont: false,
                                                })
                                            }
                                            title="Tipografia padrão"
                                        >
                                            <span className="a11y-mode-icon">
                                                Tt
                                            </span>
                                            <span className="a11y-mode-label">
                                                Padrão
                                            </span>
                                        </button>
                                        <button
                                            className={`accessibility-mode-button ${accessibility?.dyslexicFont ? "active" : ""}`}
                                            onClick={() =>
                                                saveAccessibility({
                                                    highContrast:
                                                        accessibility.highContrast,
                                                    colorBlindMode:
                                                        accessibility.colorBlindMode,
                                                    dyslexicFont: true,
                                                })
                                            }
                                            title="Ativa a fonte OpenDyslexic em títulos e textos"
                                        >
                                            <span className="a11y-mode-icon">
                                                📖
                                            </span>
                                            <span className="a11y-mode-label">
                                                Dislexia
                                            </span>
                                        </button>
                                    </div>
                                </div>

                                <div className="accessibility-section">
                                    <h3>Modo Daltonismo</h3>
                                    <div className="accessibility-mode-selector">
                                        <button
                                            className={`accessibility-mode-button ${accessibility?.colorBlindMode === "none" ? "active" : ""}`}
                                            onClick={() =>
                                                saveAccessibility({
                                                    highContrast:
                                                        accessibility.highContrast,
                                                    colorBlindMode: "none",
                                                })
                                            }
                                            title="Sem filtros de cor"
                                        >
                                            <span className="a11y-mode-icon">
                                                🌈
                                            </span>
                                            <span className="a11y-mode-label">
                                                Desativado
                                            </span>
                                        </button>
                                        <button
                                            className={`accessibility-mode-button ${accessibility?.colorBlindMode === "protanopia" ? "active" : ""}`}
                                            onClick={() =>
                                                saveAccessibility({
                                                    highContrast:
                                                        accessibility.highContrast,
                                                    colorBlindMode:
                                                        "protanopia",
                                                })
                                            }
                                            title="Filtro para dificuldade em distinguir vermelho"
                                        >
                                            <span className="a11y-mode-icon">
                                                🔴
                                            </span>
                                            <span className="a11y-mode-label">
                                                Protanopia
                                            </span>
                                        </button>
                                        <button
                                            className={`accessibility-mode-button ${accessibility?.colorBlindMode === "deuteranopia" ? "active" : ""}`}
                                            onClick={() =>
                                                saveAccessibility({
                                                    highContrast:
                                                        accessibility.highContrast,
                                                    colorBlindMode:
                                                        "deuteranopia",
                                                })
                                            }
                                            title="Filtro para dificuldade em distinguir verde"
                                        >
                                            <span className="a11y-mode-icon">
                                                🟢
                                            </span>
                                            <span className="a11y-mode-label">
                                                Deuteranopia
                                            </span>
                                        </button>
                                        <button
                                            className={`accessibility-mode-button ${accessibility?.colorBlindMode === "tritanopia" ? "active" : ""}`}
                                            onClick={() =>
                                                saveAccessibility({
                                                    highContrast:
                                                        accessibility.highContrast,
                                                    colorBlindMode:
                                                        "tritanopia",
                                                })
                                            }
                                            title="Filtro para dificuldade em distinguir azul"
                                        >
                                            <span className="a11y-mode-icon">
                                                🔵
                                            </span>
                                            <span className="a11y-mode-label">
                                                Tritanopia
                                            </span>
                                        </button>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </section>
            )}

            {/* TAB: THEMES & COLORS */}
            {activeTab === "themes" && (
                <>
                    {/* Root Only Sections */}
                    {isRoot && (
                        <>
                            {/* Theme Permissions Section */}
                            <section className="appearance-section permissions-section">
                                <h2>🔒 Permissões de Tema</h2>
                                <p className="section-description">
                                    Controle quem pode personalizar as
                                    configurações de tema. Usuários root sempre
                                    podem editar. Esta configuração está
                                    sincronizada com as permissões de cada
                                    cargo.
                                </p>

                                <div className="permission-status-buttons">
                                    <button
                                        className={`permission-status-btn allow ${permissionStatus === "allow" ? "active" : ""}`}
                                        onClick={() =>
                                            handlePermissionStatusChange(
                                                "allow",
                                            )
                                        }
                                        disabled={
                                            permissionStatus === "loading"
                                        }
                                    >
                                        ✅ Permitir
                                    </button>
                                    <button
                                        className={`permission-status-btn block ${permissionStatus === "block" ? "active" : ""}`}
                                        onClick={() =>
                                            handlePermissionStatusChange(
                                                "block",
                                            )
                                        }
                                        disabled={
                                            permissionStatus === "loading"
                                        }
                                    >
                                        🚫 Bloquear
                                    </button>
                                    <button
                                        className={`permission-status-btn custom ${permissionStatus === "custom" ? "active" : ""}`}
                                        onClick={() =>
                                        (window.location.hash =
                                            "#/settings?section=roles")
                                        }
                                        disabled={
                                            permissionStatus === "loading"
                                        }
                                    >
                                        ⚙️ Personalizado
                                    </button>
                                </div>

                                <p className="permission-status-description">
                                    {permissionStatus === "allow" && (
                                        <>
                                            <strong>Permitir:</strong> Todos os
                                            usuários podem personalizar seu tema
                                            pessoal.
                                        </>
                                    )}
                                    {permissionStatus === "block" && (
                                        <>
                                            <strong>Bloquear:</strong> Nenhum
                                            usuário (exceto root) pode alterar
                                            seu tema pessoal.
                                        </>
                                    )}
                                    {permissionStatus === "custom" && (
                                        <>
                                            <strong>Personalizado:</strong> As
                                            permissões estão configuradas por
                                            cargo. Clique no botão para
                                            gerenciar no menu de Papéis &
                                            Permissões.
                                        </>
                                    )}
                                    {permissionStatus === "loading" && (
                                        <>Carregando configurações...</>
                                    )}
                                </p>
                            </section>

                            {/* Custom Permission Warning Modal */}
                            {showCustomWarning && (
                                <div className="modal-overlay">
                                    <div className="modal-content warning-modal">
                                        <h3>⚠️ Atenção</h3>
                                        <p>
                                            Você está prestes a alterar
                                            permissões que foram configuradas
                                            individualmente por cargo. Esta ação
                                            irá sobrescrever as configurações
                                            personalizadas e aplicar a mesma
                                            permissão para todos os cargos.
                                        </p>
                                        <p>
                                            <strong>
                                                Tem certeza que deseja
                                                continuar?
                                            </strong>
                                        </p>
                                        <div className="modal-actions">
                                            <button
                                                className="modal-btn cancel"
                                                onClick={cancelCustomWarning}
                                            >
                                                Cancelar
                                            </button>
                                            <button
                                                className="modal-btn confirm"
                                                onClick={confirmCustomWarning}
                                            >
                                                Sim, continuar
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            )}

                            {/* Global Theme is now integrated in Theme and Colors */}
                        </>
                    )}

                    {/* Theme Selection Section */}
                    <section className="appearance-section appearance-section-nav-color">
                        <h2>🎨 Tema e Cores</h2>

                        {!canEditTheme && (
                            <div className="permission-warning">
                                <span className="warning-icon">⚠️</span>
                                <span>
                                    Você não tem permissão para alterar
                                    configurações de tema. Entre em contato com
                                    um administrador se precisar de acesso.
                                </span>
                            </div>
                        )}

                        {canEditTheme && (
                            <>
                                <p className="section-description">
                                    Clique em um tema para visualizar as
                                    mudanças em tempo real. As alterações são
                                    salvas automaticamente.
                                </p>

                                {isRoot ? (
                                    <div className="theme-management-container">
                                        <div className="theme-block-actions">
                                            <button
                                                type="button"
                                                className="theme-action-button apply-global-small"
                                                onClick={handleSaveGlobalTheme}
                                                title="Define o tema salvo como padrão para novos usuários"
                                            >
                                                Atualizar Tema <br></br> Padrão
                                                Global
                                            </button>
                                            <button
                                                type="button"
                                                className="theme-action-button unlock-all-small"
                                                onClick={handleUnlockAllThemes}
                                                title="Libera todos os temas"
                                            >
                                                Liberar Todos
                                            </button>
                                            <button
                                                type="button"
                                                className="theme-action-button block-all-small"
                                                onClick={
                                                    handleBlockAllExceptSelected
                                                }
                                                title="Bloqueia todos os temas exceto o selecionado"
                                            >
                                                Bloquear Todos Exceto
                                                Selecionado
                                            </button>
                                        </div>

                                        <div
                                            className="global-theme-info"
                                            style={{
                                                display: "flex",
                                                flexDirection: "column",
                                                gap: "5px",
                                            }}
                                        >
                                            <p style={{ margin: 0 }}>
                                                <strong>
                                                    Tema Padrão Global
                                                    atualmente selecionado:
                                                </strong>{" "}
                                                {getThemeName(globalTheme) ||
                                                    "N/A"}
                                            </p>
                                            <p style={{ margin: 0 }}>
                                                <strong>
                                                    Tema do Usuário Atual
                                                    atualmente selecionado:
                                                </strong>{" "}
                                                {getThemeName(
                                                    userThemeSettings?.preset ||
                                                    savedTheme?.preset ||
                                                    formData.theme?.preset,
                                                ) || "N/A"}
                                            </p>
                                        </div>

                                        <p className="theme-selection-note">
                                            Arraste os temas entre os blocos
                                            para bloquear ou liberar. Use
                                            Ctrl+Clique ou Shift+Clique para
                                            selecionar múltiplos. As alterações
                                            são salvas automaticamente.
                                        </p>

                                        <div className="theme-blocks-wrapper">
                                            <div className="theme-block">
                                                <h4>✅ Temas Liberados</h4>
                                                <div
                                                    className="preset-grid theme-drop-zone allowed-zone"
                                                    onDragOver={
                                                        handleContainerDragOver
                                                    }
                                                    onDragLeave={
                                                        handleContainerDragLeave
                                                    }
                                                    onDrop={handleDropToAllowed}
                                                    onMouseDown={(e) =>
                                                        handleLassoStart(
                                                            e,
                                                            "allowed",
                                                        )
                                                    }
                                                    onMouseMove={
                                                        handleLassoMove
                                                    }
                                                    onMouseUp={handleLassoEnd}
                                                    onMouseLeave={
                                                        handleLassoEnd
                                                    }
                                                >
                                                    {allowedThemes.includes(
                                                        "custom",
                                                    ) && (
                                                            <button
                                                                type="button"
                                                                data-theme="custom"
                                                                draggable
                                                                className={`preset-card ${savedTheme.preset ===
                                                                    "custom"
                                                                    ? "active"
                                                                    : ""
                                                                    } ${selectedThemes.has(
                                                                        "custom",
                                                                    )
                                                                        ? "selected"
                                                                        : ""
                                                                    } ${isDragging &&
                                                                        draggedThemes?.includes(
                                                                            "custom",
                                                                        )
                                                                        ? "dragging"
                                                                        : ""
                                                                    }`}
                                                                onClick={(e) => {
                                                                    if (
                                                                        e.ctrlKey ||
                                                                        e.metaKey
                                                                    ) {
                                                                        e.stopPropagation();
                                                                        toggleThemeSelection(
                                                                            "custom",
                                                                        );
                                                                    } else if (
                                                                        e.shiftKey
                                                                    ) {
                                                                        e.stopPropagation();
                                                                        setSelectionStart(
                                                                            "custom",
                                                                        );
                                                                        selectThemeRange(
                                                                            "custom",
                                                                            allowedThemes,
                                                                        );
                                                                    } else {
                                                                        handleThemeChange(
                                                                            "preset",
                                                                            "custom",
                                                                        );
                                                                    }
                                                                }}
                                                                onDragStart={(e) =>
                                                                    handleThemeDragStart(
                                                                        "custom",
                                                                        e,
                                                                    )
                                                                }
                                                                onDragEnd={() => {
                                                                    setIsDragging(
                                                                        false,
                                                                    );
                                                                }}
                                                                title="Arraste para bloquear, Ctrl+Clique para selecionar, Shift+Clique para intervalo"
                                                            >
                                                                <div
                                                                    className="preset-preview"
                                                                    style={{
                                                                        background:
                                                                            "#ccc",
                                                                    }}
                                                                ></div>
                                                                <span>
                                                                    Personalizado
                                                                </span>
                                                            </button>
                                                        )}
                                                    {Object.entries(
                                                        THEME_PRESETS,
                                                    )
                                                        .filter(([key]) =>
                                                            allowedThemes.includes(
                                                                key,
                                                            ),
                                                        )
                                                        .map(
                                                            ([key, preset]) => (
                                                                <button
                                                                    key={key}
                                                                    type="button"
                                                                    data-theme={
                                                                        key
                                                                    }
                                                                    draggable
                                                                    className={`preset-card ${savedTheme.preset ===
                                                                        key
                                                                        ? "active"
                                                                        : ""
                                                                        } ${selectedThemes.has(
                                                                            key,
                                                                        )
                                                                            ? "selected"
                                                                            : ""
                                                                        } ${isDragging &&
                                                                            draggedThemes?.includes(
                                                                                key,
                                                                            )
                                                                            ? "dragging"
                                                                            : ""
                                                                        }`}
                                                                    onClick={(
                                                                        e,
                                                                    ) => {
                                                                        if (
                                                                            e.ctrlKey ||
                                                                            e.metaKey
                                                                        ) {
                                                                            e.stopPropagation();
                                                                            toggleThemeSelection(
                                                                                key,
                                                                            );
                                                                        } else if (
                                                                            e.shiftKey
                                                                        ) {
                                                                            e.stopPropagation();
                                                                            setSelectionStart(
                                                                                key,
                                                                            );
                                                                            selectThemeRange(
                                                                                key,
                                                                                allowedThemes,
                                                                            );
                                                                        } else {
                                                                            handleThemeChange(
                                                                                "preset",
                                                                                key,
                                                                            );
                                                                        }
                                                                    }}
                                                                    onDragStart={(
                                                                        e,
                                                                    ) =>
                                                                        handleThemeDragStart(
                                                                            key,
                                                                            e,
                                                                        )
                                                                    }
                                                                    onDragEnd={() => {
                                                                        setIsDragging(
                                                                            false,
                                                                        );
                                                                    }}
                                                                    title="Arraste para bloquear, Ctrl+Clique para selecionar, Shift+Clique para intervalo"
                                                                >
                                                                    <div
                                                                        className="preset-preview"
                                                                        style={{
                                                                            background: `linear-gradient(135deg, ${preset.light.buttonSecondary} 50%, ${preset.light.buttonPrimary} 50%)`,
                                                                        }}
                                                                    ></div>
                                                                    <span>
                                                                        {
                                                                            preset.name
                                                                        }
                                                                    </span>
                                                                </button>
                                                            ),
                                                        )}
                                                    {allowedThemes.length ===
                                                        0 && (
                                                            <div className="preset-empty">
                                                                Nenhum tema liberado
                                                            </div>
                                                        )}
                                                    {isLassoSelecting &&
                                                        lassoRect &&
                                                        currentZone ===
                                                        "allowed" && (
                                                            <div
                                                                className="lasso-selection-box"
                                                                style={{
                                                                    left: `${lassoRect.left}px`,
                                                                    top: `${lassoRect.top}px`,
                                                                    width: `${lassoRect.right - lassoRect.left}px`,
                                                                    height: `${lassoRect.bottom - lassoRect.top}px`,
                                                                }}
                                                            />
                                                        )}
                                                </div>
                                            </div>

                                            <div className="theme-block">
                                                <h4>🚫 Temas Bloqueados</h4>
                                                <div
                                                    className="preset-grid theme-drop-zone blocked-zone"
                                                    onDragOver={
                                                        handleContainerDragOver
                                                    }
                                                    onDragLeave={
                                                        handleContainerDragLeave
                                                    }
                                                    onDrop={handleDropToBlocked}
                                                    onMouseDown={(e) =>
                                                        handleLassoStart(
                                                            e,
                                                            "blocked",
                                                        )
                                                    }
                                                    onMouseMove={
                                                        handleLassoMove
                                                    }
                                                    onMouseUp={handleLassoEnd}
                                                    onMouseLeave={
                                                        handleLassoEnd
                                                    }
                                                >
                                                    {blockedThemes.map(
                                                        (key) => {
                                                            const preset =
                                                                key === "custom"
                                                                    ? null
                                                                    : THEME_PRESETS[
                                                                    key
                                                                    ];
                                                            return (
                                                                <button
                                                                    key={key}
                                                                    type="button"
                                                                    data-theme={
                                                                        key
                                                                    }
                                                                    draggable
                                                                    className={`preset-card ${selectedThemes.has(
                                                                        key,
                                                                    )
                                                                        ? "selected"
                                                                        : ""
                                                                        } ${isDragging &&
                                                                            draggedThemes?.includes(
                                                                                key,
                                                                            )
                                                                            ? "dragging"
                                                                            : ""
                                                                        }`}
                                                                    onClick={(
                                                                        e,
                                                                    ) => {
                                                                        if (
                                                                            e.ctrlKey ||
                                                                            e.metaKey
                                                                        ) {
                                                                            e.stopPropagation();
                                                                            toggleThemeSelection(
                                                                                key,
                                                                            );
                                                                        } else if (
                                                                            e.shiftKey
                                                                        ) {
                                                                            e.stopPropagation();
                                                                            setSelectionStart(
                                                                                key,
                                                                            );
                                                                            selectThemeRange(
                                                                                key,
                                                                                blockedThemes,
                                                                            );
                                                                        } else {
                                                                            setSelectionStart(
                                                                                key,
                                                                            );
                                                                            setSelectedThemes(
                                                                                new Set(
                                                                                    [
                                                                                        key,
                                                                                    ],
                                                                                ),
                                                                            );
                                                                        }
                                                                    }}
                                                                    onDragStart={(
                                                                        e,
                                                                    ) =>
                                                                        handleThemeDragStart(
                                                                            key,
                                                                            e,
                                                                        )
                                                                    }
                                                                    onDragEnd={() => {
                                                                        setIsDragging(
                                                                            false,
                                                                        );
                                                                    }}
                                                                    title="Arraste para liberar, Ctrl+Clique para selecionar, Shift+Clique para intervalo"
                                                                >
                                                                    <div
                                                                        className="preset-preview"
                                                                        style={{
                                                                            background:
                                                                                key ===
                                                                                    "custom"
                                                                                    ? "#ccc"
                                                                                    : `linear-gradient(135deg, ${preset.light.buttonSecondary} 50%, ${preset.light.buttonPrimary} 50%)`,
                                                                        }}
                                                                    ></div>
                                                                    <span>
                                                                        {key ===
                                                                            "custom"
                                                                            ? "Personalizado"
                                                                            : preset?.name ||
                                                                            key}
                                                                    </span>
                                                                </button>
                                                            );
                                                        },
                                                    )}
                                                    {blockedThemes.length ===
                                                        0 && (
                                                            <div className="preset-empty">
                                                                Nenhum tema
                                                                bloqueado
                                                            </div>
                                                        )}
                                                    {isLassoSelecting &&
                                                        lassoRect &&
                                                        currentZone ===
                                                        "blocked" && (
                                                            <div
                                                                className="lasso-selection-box"
                                                                style={{
                                                                    left: `${lassoRect.left}px`,
                                                                    top: `${lassoRect.top}px`,
                                                                    width: `${lassoRect.right - lassoRect.left}px`,
                                                                    height: `${lassoRect.bottom - lassoRect.top}px`,
                                                                }}
                                                            />
                                                        )}
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                ) : (
                                    <div className="form-group">
                                        <label>Predefinição de Tema</label>
                                        <div className="preset-grid">
                                            {allowedThemes.includes(
                                                "custom",
                                            ) && (
                                                    <button
                                                        type="button"
                                                        className={`preset-card ${savedTheme.preset === "custom" ? "active" : ""}`}
                                                        onClick={() =>
                                                            handleThemeChange(
                                                                "preset",
                                                                "custom",
                                                            )
                                                        }
                                                    >
                                                        <div
                                                            className="preset-preview"
                                                            style={{
                                                                background: "#ccc",
                                                            }}
                                                        ></div>
                                                        <span>Personalizado</span>
                                                    </button>
                                                )}
                                            {Object.entries(THEME_PRESETS)
                                                .filter(([key]) =>
                                                    allowedThemes.includes(key),
                                                )
                                                .map(([key, preset]) => (
                                                    <button
                                                        key={key}
                                                        type="button"
                                                        className={`preset-card ${savedTheme.preset === key ? "active" : ""}`}
                                                        onClick={() =>
                                                            handleThemeChange(
                                                                "preset",
                                                                key,
                                                            )
                                                        }
                                                    >
                                                        <div
                                                            className="preset-preview"
                                                            style={{
                                                                background: `linear-gradient(135deg, ${preset.light.buttonSecondary} 50%, ${preset.light.buttonPrimary} 50%)`,
                                                            }}
                                                        ></div>
                                                        <span>
                                                            {preset.name}
                                                        </span>
                                                    </button>
                                                ))}
                                        </div>
                                    </div>
                                )}

                                {formData.theme?.preset === "custom" && (
                                    <>
                                        <div className="form-row">
                                            <div className="form-group">
                                                <label>Cor Primária</label>
                                                <div className="color-input-wrapper">
                                                    <input
                                                        type="color"
                                                        value={
                                                            formData.theme
                                                                ?.primaryColor ||
                                                            "#3498db"
                                                        }
                                                        onChange={(e) =>
                                                            handleThemeChange(
                                                                "primaryColor",
                                                                e.target.value,
                                                            )
                                                        }
                                                    />
                                                    <input
                                                        type="text"
                                                        value={
                                                            formData.theme
                                                                ?.primaryColor ||
                                                            ""
                                                        }
                                                        onChange={(e) =>
                                                            handleThemeChange(
                                                                "primaryColor",
                                                                e.target.value,
                                                            )
                                                        }
                                                    />
                                                </div>
                                            </div>
                                            <div className="form-group">
                                                <label>Cor Secundária</label>
                                                <div className="color-input-wrapper">
                                                    <input
                                                        type="color"
                                                        value={
                                                            formData.theme
                                                                ?.secondaryColor ||
                                                            "#2c3e50"
                                                        }
                                                        onChange={(e) =>
                                                            handleThemeChange(
                                                                "secondaryColor",
                                                                e.target.value,
                                                            )
                                                        }
                                                    />
                                                    <input
                                                        type="text"
                                                        value={
                                                            formData.theme
                                                                ?.secondaryColor ||
                                                            ""
                                                        }
                                                        onChange={(e) =>
                                                            handleThemeChange(
                                                                "secondaryColor",
                                                                e.target.value,
                                                            )
                                                        }
                                                    />
                                                </div>
                                            </div>
                                        </div>
                                        <div className="form-row">
                                            <div className="form-group">
                                                <label>Cor de Fundo</label>
                                                <div className="color-input-wrapper">
                                                    <input
                                                        type="color"
                                                        value={
                                                            formData.theme
                                                                ?.backgroundColor ||
                                                            "#f8f9fa"
                                                        }
                                                        onChange={(e) =>
                                                            handleThemeChange(
                                                                "backgroundColor",
                                                                e.target.value,
                                                            )
                                                        }
                                                    />
                                                    <input
                                                        type="text"
                                                        value={
                                                            formData.theme
                                                                ?.backgroundColor ||
                                                            ""
                                                        }
                                                        onChange={(e) =>
                                                            handleThemeChange(
                                                                "backgroundColor",
                                                                e.target.value,
                                                            )
                                                        }
                                                    />
                                                </div>
                                            </div>
                                            <div className="form-group">
                                                <label>Cor de Superfície</label>
                                                <div className="color-input-wrapper">
                                                    <input
                                                        type="color"
                                                        value={
                                                            formData.theme
                                                                ?.surfaceColor ||
                                                            "#ffffff"
                                                        }
                                                        onChange={(e) =>
                                                            handleThemeChange(
                                                                "surfaceColor",
                                                                e.target.value,
                                                            )
                                                        }
                                                    />
                                                    <input
                                                        type="text"
                                                        value={
                                                            formData.theme
                                                                ?.surfaceColor ||
                                                            ""
                                                        }
                                                        onChange={(e) =>
                                                            handleThemeChange(
                                                                "surfaceColor",
                                                                e.target.value,
                                                            )
                                                        }
                                                    />
                                                </div>
                                            </div>
                                        </div>
                                        <div className="form-row">
                                            <div className="form-group">
                                                <label>
                                                    Cor do Texto Principal
                                                </label>
                                                <div className="color-input-wrapper">
                                                    <input
                                                        type="color"
                                                        value={
                                                            formData.theme
                                                                ?.textColor ||
                                                            "#333333"
                                                        }
                                                        onChange={(e) =>
                                                            handleThemeChange(
                                                                "textColor",
                                                                e.target.value,
                                                            )
                                                        }
                                                    />
                                                    <input
                                                        type="text"
                                                        value={
                                                            formData.theme
                                                                ?.textColor ||
                                                            ""
                                                        }
                                                        onChange={(e) =>
                                                            handleThemeChange(
                                                                "textColor",
                                                                e.target.value,
                                                            )
                                                        }
                                                    />
                                                </div>
                                            </div>
                                            <div className="form-group">
                                                <label>
                                                    Cor do Texto Secundário
                                                </label>
                                                <div className="color-input-wrapper">
                                                    <input
                                                        type="color"
                                                        value={
                                                            formData.theme
                                                                ?.textSecondaryColor ||
                                                            "#666666"
                                                        }
                                                        onChange={(e) =>
                                                            handleThemeChange(
                                                                "textSecondaryColor",
                                                                e.target.value,
                                                            )
                                                        }
                                                    />
                                                    <input
                                                        type="text"
                                                        value={
                                                            formData.theme
                                                                ?.textSecondaryColor ||
                                                            ""
                                                        }
                                                        onChange={(e) =>
                                                            handleThemeChange(
                                                                "textSecondaryColor",
                                                                e.target.value,
                                                            )
                                                        }
                                                    />
                                                </div>
                                            </div>
                                        </div>
                                        <div className="form-row">
                                            <div className="form-group">
                                                <label>Cor da Borda</label>
                                                <div className="color-input-wrapper">
                                                    <input
                                                        type="color"
                                                        value={
                                                            formData.theme
                                                                ?.borderColor ||
                                                            "#cbd5e0"
                                                        }
                                                        onChange={(e) =>
                                                            handleThemeChange(
                                                                "borderColor",
                                                                e.target.value,
                                                            )
                                                        }
                                                    />
                                                    <input
                                                        type="text"
                                                        value={
                                                            formData.theme
                                                                ?.borderColor ||
                                                            ""
                                                        }
                                                        onChange={(e) =>
                                                            handleThemeChange(
                                                                "borderColor",
                                                                e.target.value,
                                                            )
                                                        }
                                                    />
                                                </div>
                                            </div>
                                        </div>
                                    </>
                                )}
                            </>
                        )}
                    </section>
                </>
            )}
        </div>
    );
}
