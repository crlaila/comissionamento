import React, { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import './Layout.css'

interface LayoutProps {
  children: React.ReactNode
}

export const Layout: React.FC<LayoutProps> = ({ children }) => {
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  const getRoleLabel = (role: string): string => {
    const roleLabels: Record<string, string> = {
      rep: 'Representante',
      manager: 'Gerenciador',
      finance: 'Financeiro',
      admin: 'Administrador',
    }
    return roleLabels[role] || role
  }

  const getNavigationItems = () => {
    const baseItems = [
      { label: 'Dashboard', path: '/' },
    ]

    if (user?.role === 'manager' || user?.role === 'admin') {
      baseItems.push({ label: 'Equipe', path: '/team' })
    }

    if (user?.role === 'finance' || user?.role === 'admin') {
      baseItems.push({ label: 'Relatórios', path: '/reports' })
      baseItems.push({ label: 'Aprovações', path: '/approvals' })
    }

    if (user?.role === 'admin') {
      baseItems.push({ label: 'Configurações', path: '/admin' })
    }

    return baseItems
  }

  const navItems = getNavigationItems()

  return (
    <div className="layout">
      <header className="layout-header">
        <div className="header-content">
          <button
            className="sidebar-toggle"
            onClick={() => setSidebarOpen(!sidebarOpen)}
            aria-label="Toggle sidebar"
          >
            ☰
          </button>
          <h1 className="header-title">Comissionamento</h1>
          <div className="user-section">
            <span className="user-name">{user?.name}</span>
            <span className="user-role">{getRoleLabel(user?.role || '')}</span>
          </div>
        </div>
      </header>

      <div className="layout-container">
        <aside className={`layout-sidebar ${sidebarOpen ? 'open' : ''}`}>
          <nav className="sidebar-nav">
            {navItems.map((item) => (
              <Link
                key={item.path}
                to={item.path}
                className="nav-item"
                onClick={() => setSidebarOpen(false)}
              >
                {item.label}
              </Link>
            ))}
          </nav>
          <div className="sidebar-footer">
            <button
              className="logout-button"
              onClick={handleLogout}
            >
              Sair
            </button>
          </div>
        </aside>

        <main className="layout-content">
          {children}
        </main>
      </div>
    </div>
  )
}
