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
import { useNavigate } from "react-router-dom";
import Select from "react-select";
import { useConfig } from "../contexts/ConfigContext";
import { usersApi } from "../api/usersApi";
import { useUrlState } from "../hooks/useUrlState";
import {
    filterUsers,
    getInitialFormData,
    formatUserForEdit,
} from "../utils/userHelpers";
import UsersTable from "../components/users/UsersTable";
import UserModal from "../components/users/UserModal";
import PrimaryButton from "../components/common/PrimaryButton";
import "./styles/Users.css";

export default function Users({
    token,
    apiUrl,
    user,
    onLogout,
    onTokenExpired,
}) {
    // All hooks must be called at the top level, before any early returns
    const { config, getGenderHelpers } = useConfig();
    const navigate = useNavigate();
    const g = getGenderHelpers("user");

    const [users, setUsers] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");
    const [modalError, setModalError] = useState("");
    const [success, setSuccess] = useState("");

    // State persistence for filters
    const { values, updateValue } = useUrlState(
        { username: "", displayName: "", role: "" },
        { debounce: true, debounceTime: 300, syncWithUrl: false },
    );
    const usernameFilter = values.username || "";
    const displayNameFilter = values.displayName || "";
    const roleFilter = values.role || "";
    const setUsernameFilter = (val) => updateValue("username", val);
    const setDisplayNameFilter = (val) => updateValue("displayName", val);
    const setRoleFilter = (val) => updateValue("role", val);

    const [showModal, setShowModal] = useState(false);
    const [modalMode, setModalMode] = useState("create");
    const [selectedUser, setSelectedUser] = useState(null);
    const [formData, setFormData] = useState(getInitialFormData());

    // StrictMode-safe guard: prevent double-firing of initial load
    const usersLoadStarted = useRef(false);
    useEffect(() => {
        if (usersLoadStarted.current) return;
        usersLoadStarted.current = true;
        loadUsers();
    }, []);

    const loadUsers = async () => {
        setLoading(true);
        setError("");
        try {
            const data = await usersApi.loadUsers(
                apiUrl,
                token,
                onTokenExpired,
            );
            setUsers(data);
        } catch (err) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    };

    const handleCreateUser = async () => {
        // ... rest of functions unchanged until return ...
        setModalError("");
        try {
            await usersApi.createUser(apiUrl, token, formData, onTokenExpired);
            await loadUsers();
            closeModal();
            setSuccess("Usuário criado com sucesso!");
            setTimeout(() => setSuccess(""), 3000);
        } catch (err) {
            setModalError(err.message);
        }
    };

    const handleUpdateUser = async () => {
        setModalError("");
        try {
            const payload = {
                display_name: formData.display_name,
                role: formData.role,
            };

            if (formData.password) {
                payload.password = formData.password;
            }

            if (
                formData.username &&
                formData.username !== selectedUser.username
            ) {
                payload.username = formData.username;
            }

            const isUpdatingSelf = selectedUser.username === user.username;

            const response = await usersApi.updateUser(
                apiUrl,
                token,
                selectedUser.username,
                payload,
                onTokenExpired,
            );

            // Se o usuário alterou a própria senha, deslogar
            if (
                isUpdatingSelf &&
                response.data &&
                response.data.logout_required
            ) {
                closeModal();
                setSuccess(response.message || "Senha atualizada com sucesso!");
                setTimeout(() => {
                    alert(
                        "Sua senha foi alterada. Você será deslogado e precisará fazer login novamente.",
                    );
                    onLogout();
                }, 1500);
                return;
            }

            await loadUsers();
            closeModal();
            setSuccess("Usuário atualizado com sucesso!");
            setTimeout(() => setSuccess(""), 3000);
        } catch (err) {
            setModalError(err.message);
        }
    };

    const handleBlockUser = async (username) => {
        setError("");
        setSuccess("");

        if (username === user.username) {
            setError("Você não pode bloquear a si mesmo!");
            return;
        }

        if (
            !window.confirm(
                `Tem certeza que deseja bloquear o usuário ${username}?`,
            )
        ) {
            return;
        }

        try {
            await usersApi.blockUser(apiUrl, token, username, onTokenExpired);
            await loadUsers();
            setSuccess(`Usuário ${username} bloqueado com sucesso!`);
            setTimeout(() => setSuccess(""), 3000);
        } catch (err) {
            setError(err.message);
        }
    };

    const handleUnlockUser = async (username) => {
        setError("");
        setSuccess("");

        try {
            await usersApi.unlockUser(apiUrl, token, username, onTokenExpired);
            await loadUsers();
            setSuccess(`Usuário ${username} desbloqueado com sucesso!`);
            setTimeout(() => setSuccess(""), 3000);
        } catch (err) {
            setError(err.message);
        }
    };

    const handleDeleteUser = async (username) => {
        setError("");
        setSuccess("");

        if (username === user.username) {
            setError("Você não pode deletar a si mesmo!");
            return;
        }

        if (
            !window.confirm(
                `Tem certeza que deseja DELETAR permanentemente o usuário ${username}?\n\nEsta ação não pode ser desfeita!`,
            )
        ) {
            return;
        }

        try {
            await usersApi.deleteUser(apiUrl, token, username, onTokenExpired);
            await loadUsers();
            setSuccess(`Usuário ${username} deletado com sucesso!`);
            setTimeout(() => setSuccess(""), 3000);
        } catch (err) {
            setError(err.message);
        }
    };

    const openCreateModal = () => {
        setModalMode("create");
        setFormData(getInitialFormData());
        setShowModal(true);
    };

    const openEditModal = (userToEdit) => {
        setModalMode("edit");
        setSelectedUser(userToEdit);
        setFormData(formatUserForEdit(userToEdit));
        setShowModal(true);
    };

    const closeModal = () => {
        setShowModal(false);
        setSelectedUser(null);
        setFormData(getInitialFormData());
        setModalError("");
    };

    const handleSubmit = (e) => {
        e.preventDefault();
        if (modalMode === "create") {
            handleCreateUser();
        } else {
            handleUpdateUser();
        }
    };

    function compareAlphaNum(a, b) {
        const regex = /(.*?)(\d+)$/;
        const aMatch = (a.username || "").match(regex);
        const bMatch = (b.username || "").match(regex);

        if (aMatch && bMatch && aMatch[1] === bMatch[1]) {
            // Se prefixo igual, compara número como inteiro
            return parseInt(aMatch[2], 10) - parseInt(bMatch[2], 10);
        }
        // Caso contrário, ordena normalmente
        return (a.username || "").localeCompare(b.username || "");
    }

    const filteredUsers = filterUsers([...users].sort(compareAlphaNum), {
        username: usernameFilter,
        displayName: displayNameFilter,
        role: roleFilter,
    });

    return (
        <div className="users-container">
            <div className="users-header">
                <h1 className="users-title">
                    👤 {config.labels.users || "Usuários"}
                </h1>
                <div className="button-group">
                    {user?.resources?.['roles']?.includes('read') && (
                        <PrimaryButton
                            onClick={() => navigate("/settings?section=roles")}
                            style={{ minWidth: "auto" }}
                            title="Gerenciar papéis e permissões"
                        >
                            👨‍💼 Papéis & Permissões
                        </PrimaryButton>
                    )}
                    {user?.resources?.['users']?.includes('create') && (
                        <PrimaryButton onClick={openCreateModal}>
                            + {g.new} {config.labels.user || "Usuário"}
                        </PrimaryButton>
                    )}
                </div>
            </div>

            {error && <div className="users-error">{error}</div>}

            {success && <div className="users-success">{success}</div>}

            <div className="users-search-container">
                <Select
                    value={
                        usernameFilter
                            ? { value: usernameFilter, label: usernameFilter }
                            : null
                    }
                    onChange={(selected) =>
                        setUsernameFilter(selected ? selected.value : "")
                    }
                    options={[
                        { value: "", label: "Todos os usuários" },
                        ...[...new Set(users.map((u) => u.username))].map(
                            (username) => ({
                                value: username,
                                label: username,
                            }),
                        ),
                    ]}
                    isSearchable={true}
                    placeholder="Filtrar por nome de usuário..."
                    styles={{
                        control: (provided) => ({
                            ...provided,
                            minWidth: "200px",
                            fontSize: "14px",
                        }),
                    }}
                />

                <Select
                    value={
                        displayNameFilter
                            ? {
                                value: displayNameFilter,
                                label: displayNameFilter,
                            }
                            : null
                    }
                    onChange={(selected) =>
                        setDisplayNameFilter(selected ? selected.value : "")
                    }
                    options={[
                        { value: "", label: "Todos os nomes de exibição" },
                        ...[...new Set(users.map((u) => u.display_name))].map(
                            (displayName) => ({
                                value: displayName,
                                label: displayName,
                            }),
                        ),
                    ]}
                    isSearchable={true}
                    placeholder="Filtrar por nome de exibição..."
                    styles={{
                        control: (provided) => ({
                            ...provided,
                            minWidth: "200px",
                            fontSize: "14px",
                        }),
                    }}
                />

                <Select
                    value={
                        roleFilter
                            ? { value: roleFilter, label: roleFilter }
                            : null
                    }
                    onChange={(selected) =>
                        setRoleFilter(selected ? selected.value : "")
                    }
                    options={[
                        { value: "", label: "Todas as funções" },
                        { value: "root", label: "Root" },
                        { value: "admin", label: "Administrador" },
                        { value: "user", label: "Usuário" },
                    ]}
                    isSearchable={false}
                    placeholder="Filtrar por função..."
                    styles={{
                        control: (provided) => ({
                            ...provided,
                            minWidth: "200px",
                            fontSize: "14px",
                        }),
                    }}
                />

                {(usernameFilter || displayNameFilter || roleFilter) && (
                    <button
                        onClick={() => {
                            setUsernameFilter("");
                            setDisplayNameFilter("");
                            setRoleFilter("");
                        }}
                        className="users-clear-filters-button"
                        title="Limpar filtros"
                    >
                        ✕ Limpar
                    </button>
                )}
            </div>

            <div className="users-table-wrapper">
                {/* <div className="users-table-header">
                    <h2 className="users-table-header-title">Usuários</h2>
                </div>*/}
                <UsersTable
                    filteredUsers={filteredUsers}
                    currentUser={user}
                    onEdit={openEditModal}
                    onBlock={handleBlockUser}
                    onUnlock={handleUnlockUser}
                    onDelete={handleDeleteUser}
                />
            </div>

            <UserModal
                showModal={showModal}
                modalMode={modalMode}
                formData={formData}
                setFormData={setFormData}
                onSubmit={handleSubmit}
                onClose={closeModal}
                error={modalError}
                currentUserRole={user.role}
                apiUrl={apiUrl}
                token={token}
            />

            {/* Exemplo de uso: AuditLogsTable recebe lista de usuários */}
            {/* <AuditLogsTable logs={logs} users={users} ... /> */}
        </div>
    );
}
