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
import "./styles/Initialize.css";

const generateSecurePassword = (length = 64) => {
    const uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ";
    const lowercase = "abcdefghijklmnopqrstuvwxyz";
    const numbers = "0123456789";
    const symbols = "!@#$%^&*()-_=+[]{}.,;:<>?";
    const allChars = uppercase + lowercase + numbers + symbols;

    let newPassword = "";
    // Ensure at least one of each type
    newPassword += uppercase.charAt(
        Math.floor(Math.random() * uppercase.length),
    );
    newPassword += lowercase.charAt(
        Math.floor(Math.random() * lowercase.length),
    );
    newPassword += numbers.charAt(Math.floor(Math.random() * numbers.length));
    newPassword += symbols.charAt(Math.floor(Math.random() * symbols.length));

    // Fill the rest randomly
    for (let i = newPassword.length; i < length; i++) {
        newPassword += allChars.charAt(
            Math.floor(Math.random() * allChars.length),
        );
    }

    // Shuffle the password
    return newPassword
        .split("")
        .sort(() => Math.random() - 0.5)
        .join("");
};

const Initialize = ({ onInitializationComplete }) => {
    const [step, setStep] = useState("check"); // check, create-admin, success
    const [status, setStatus] = useState(null);
    const [errors, setErrors] = useState([]);
    const [loading, setLoading] = useState(true);

    const PASSWORD_MIN = 32;
    const PASSWORD_MAX = 128;

    // Root user configuration
    const [username, setUsername] = useState("root");
    const [displayName, setDisplayName] = useState("Administrator");
    const [passwordLength, setPasswordLength] = useState(64);
    const [password, setPassword] = useState(generateSecurePassword(64));

    const [showPassword, setShowPassword] = useState(false);
    const [creatingAdmin, setCreatingAdmin] = useState(false);

    // StrictMode-safe guard: prevent double-firing of initial check
    const initCheckStarted = useRef(false);
    useEffect(() => {
        if (initCheckStarted.current) return;
        initCheckStarted.current = true;
        checkBackendStatus();
    }, []);

    const checkBackendStatus = async () => {
        try {
            setLoading(true);
            setErrors([]);

            // Check backend health
            const healthResponse = await fetch("/health", {
                method: "GET",
            });

            if (!healthResponse.ok) {
                throw new Error("Backend is not responding");
            }

            // Check initialization status
            const statusResponse = await fetch("/api/initialize/status", {
                method: "GET",
            });

            if (!statusResponse.ok) {
                throw new Error("Cannot check initialization status");
            }

            const statusData = await statusResponse.json();

            // If already initialized, skip this screen
            if (statusData.is_initialized) {
                onInitializationComplete();
                return;
            }

            // Proceed to admin creation
            setTimeout(() => {
                setStep("create-admin");
            }, 300);
        } catch (error) {
            setErrors([error.message]);
            setStatus({
                type: "error",
                message: "Backend server is not running",
                advice: "Ensure the backend server is running. Contact your administrator if the issue persists.",
            });
            setStep("error");
        } finally {
            setLoading(false);
        }
    };

    const handleGeneratePassword = () => {
        const newPassword = generateSecurePassword(passwordLength);
        setPassword(newPassword);
    };

    const handlePasswordLengthChange = (newLength) => {
        setPasswordLength(newLength);
        const newPassword = generateSecurePassword(newLength);
        setPassword(newPassword);
    };

    const copyToClipboard = (text) => {
        navigator.clipboard
            .writeText(text)
            .then(() => {
                alert("✓ Copied to clipboard!");
            })
            .catch(() => {
                alert("Failed to copy");
            });
    };

    const handleCreateRootAdmin = async () => {
        try {
            setCreatingAdmin(true);
            setErrors([]);

            // Validate input
            if (!username.trim()) {
                setErrors(["Username is required"]);
                return;
            }

            if (!displayName.trim()) {
                setErrors(["Display name is required"]);
                return;
            }

            if (!password || password.length < PASSWORD_MIN) {
                setErrors([
                    `Password must be at least ${PASSWORD_MIN} characters`,
                ]);
                return;
            }

            // Create root admin user
            const response = await fetch("/api/initialize/admin", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    username: username.trim(),
                    display_name: displayName.trim(),
                    password: password,
                }),
            });

            const data = await response.json();

            if (!response.ok || !data.success) {
                setErrors(data.errors || ["Failed to create root admin user"]);
                return;
            }

            // Auto copy password
            try {
                await navigator.clipboard.writeText(password);
                alert("Password copied to clipboard!");
            } catch (err) {
                console.error("Failed to auto-copy password:", err);
            }

            setStatus({
                type: "success",
                message: "✅ Root admin user created successfully!",
                credentials: {
                    username: username.trim(),
                    password: password,
                },
            });

            // Show success screen for a very short time then redirect
            setTimeout(() => {
                onInitializationComplete();
            }, 500);

            setStep("success");
        } catch (error) {
            setErrors([`Creation failed: ${error.message}`]);
        } finally {
            setCreatingAdmin(false);
        }
    };

    const handleRetry = () => {
        setStep("check");
        setErrors([]);
        setStatus(null);
        checkBackendStatus();
    };

    // ============= RENDER SCREENS =============

    if (step === "error") {
        return (
            <div className="initialize-container">
                <div className="initialize-screen error-screen">
                    <div className="init-icon">❌</div>
                    <h1>Cannot Connect to Backend</h1>
                    <p className="init-message">{status?.message}</p>
                    <p className="init-advice">{status?.advice}</p>
                    <button
                        className="init-button primary"
                        onClick={handleRetry}
                    >
                        🔄 Retry Connection
                    </button>
                </div>
            </div>
        );
    }

    if (step === "check" || loading) {
        return (
            <div className="initialize-container">
                <div className="initialize-screen checking-screen">
                    <div className="init-spinner">⏳</div>
                    <h1>Initializing Application</h1>
                    <p>Checking backend server...</p>
                    <div className="init-progress">
                        <div className="progress-item">
                            <span className="progress-icon">✓</span>
                            <span>Backend Server</span>
                            <span className="status checking">Checking...</span>
                        </div>
                    </div>
                </div>
            </div>
        );
    }

    if (step === "create-admin") {
        return (
            <div className="initialize-container">
                <div className="initialize-screen admin-creation-screen">
                    <div className="init-icon">👤</div>
                    <h1>Create Root Administrator</h1>
                    <p className="init-message">
                        Set up your root administrator account to get started
                    </p>

                    {errors.length > 0 && (
                        <div className="error-alert">
                            <strong>⚠️ Error:</strong>
                            <ul>
                                {errors.map((err, idx) => (
                                    <li key={idx}>{err}</li>
                                ))}
                            </ul>
                        </div>
                    )}

                    <div className="admin-form">
                        <div className="form-section">
                            <div className="form-group">
                                <label htmlFor="username">Username</label>
                                <input
                                    id="username"
                                    type="text"
                                    value={username}
                                    onChange={(e) =>
                                        setUsername(e.target.value)
                                    }
                                    placeholder="root"
                                    maxLength="50"
                                />
                            </div>

                            <div className="form-group">
                                <label htmlFor="display-name">
                                    Display Name
                                </label>
                                <input
                                    id="display-name"
                                    type="text"
                                    value={displayName}
                                    onChange={(e) =>
                                        setDisplayName(e.target.value)
                                    }
                                    placeholder="Administrator"
                                    maxLength="100"
                                />
                            </div>

                            <div className="form-group">
                                <div className="form-header">
                                    <label htmlFor="password">
                                        Password (Length: {passwordLength})
                                    </label>
                                    <button
                                        type="button"
                                        className="toggle-secret"
                                        onClick={() =>
                                            setShowPassword(!showPassword)
                                        }
                                        title="Show/hide password"
                                    >
                                        {showPassword ? "🙈" : "👁️"}
                                    </button>
                                </div>

                                <div className="password-input-wrapper">
                                    <input
                                        id="password"
                                        type={
                                            showPassword ? "text" : "password"
                                        }
                                        value={password}
                                        onChange={(e) =>
                                            setPassword(e.target.value)
                                        }
                                        placeholder="Enter a strong password"
                                    />
                                    <button
                                        type="button"
                                        className="copy-btn"
                                        onClick={() =>
                                            copyToClipboard(password)
                                        }
                                        title="Copy to clipboard"
                                    >
                                        📋
                                    </button>
                                </div>

                                <div className="slider-container">
                                    <input
                                        type="range"
                                        min={PASSWORD_MIN}
                                        max={PASSWORD_MAX}
                                        value={passwordLength}
                                        onChange={(e) =>
                                            handlePasswordLengthChange(
                                                parseInt(e.target.value),
                                            )
                                        }
                                        className="length-slider"
                                        title="Adjust password length"
                                    />
                                    <span className="slider-label">
                                        {PASSWORD_MIN} - {PASSWORD_MAX}
                                    </span>
                                </div>

                                <button
                                    type="button"
                                    className="generate-btn"
                                    onClick={handleGeneratePassword}
                                    title="Generate secure password"
                                >
                                    🔐 Generate Password
                                </button>

                                <small className="password-hint">
                                    💡 Use the generator to create a strong,
                                    secure password
                                </small>
                            </div>
                        </div>

                        <div className="form-section info-box">
                            <h3>📌 Important</h3>
                            <ul>
                                <li>Save this password in a secure location</li>
                                <li>
                                    You can change it later from the dashboard
                                </li>
                                <li>
                                    This account has full administrator access
                                </li>
                            </ul>
                        </div>

                        <button
                            className="init-button primary large"
                            onClick={handleCreateRootAdmin}
                            disabled={
                                creatingAdmin ||
                                !username.trim() ||
                                !displayName.trim() ||
                                password.length < PASSWORD_MIN
                            }
                            title={
                                password.length < PASSWORD_MIN
                                    ? `Password must be at least ${PASSWORD_MIN} characters`
                                    : "Create root administrator account"
                            }
                        >
                            {creatingAdmin
                                ? "⏳ Creating..."
                                : "✅ Create Administrator"}
                        </button>
                    </div>
                </div>
            </div>
        );
    }

    if (step === "success") {
        return (
            <div className="initialize-container">
                <div className="initialize-screen success-screen">
                    <div className="init-spinner">⏳</div>
                    <h1>Validating Root...</h1>
                </div>
            </div>
        );
    }

    return null;
};

export default Initialize;
