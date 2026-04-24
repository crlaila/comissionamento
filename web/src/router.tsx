import { createBrowserRouter } from 'react-router-dom'
import type { RouteObject } from 'react-router-dom'
import { Login } from './pages/Login'
import { ProtectedRoute } from './components/ProtectedRoute'
import { Layout } from './components/Layout'
import { RepDashboard } from './pages/RepDashboard'
import { TeamDashboard } from './pages/TeamDashboard'
import { OrgDashboard } from './pages/OrgDashboard'
import { GoalManagement } from './pages/GoalManagement'
import { EventDetail } from './pages/EventDetail'

// Placeholder components
const ApprovalsPage = () => <div><h2>Aprovações</h2><p>Aprovações de declarações</p></div>
const AdminPage = () => <div><h2>Administração</h2><p>Painel de administração</p></div>

const DashboardRouter = () => {
  const getDashboard = () => {
    // This will be determined based on user role via the Auth context
    // For now, we're using RepDashboard as default
    return <RepDashboard />
  }
  return getDashboard()
}

const routes: RouteObject[] = [
  {
    path: '/login',
    element: <Login />,
  },
  {
    path: '/',
    element: (
      <ProtectedRoute>
        <Layout>
          <DashboardRouter />
        </Layout>
      </ProtectedRoute>
    ),
  },
  {
    path: '/team',
    element: (
      <ProtectedRoute>
        <Layout>
          <TeamDashboard />
        </Layout>
      </ProtectedRoute>
    ),
  },
  {
    path: '/reports',
    element: (
      <ProtectedRoute>
        <Layout>
          <OrgDashboard />
        </Layout>
      </ProtectedRoute>
    ),
  },
  {
    path: '/approvals',
    element: (
      <ProtectedRoute>
        <Layout>
          <ApprovalsPage />
        </Layout>
      </ProtectedRoute>
    ),
  },
  {
    path: '/goals',
    element: (
      <ProtectedRoute>
        <Layout>
          <GoalManagement />
        </Layout>
      </ProtectedRoute>
    ),
  },
  {
    path: '/events/:eventId',
    element: (
      <ProtectedRoute>
        <Layout>
          <EventDetail />
        </Layout>
      </ProtectedRoute>
    ),
  },
  {
    path: '/admin',
    element: (
      <ProtectedRoute>
        <Layout>
          <AdminPage />
        </Layout>
      </ProtectedRoute>
    ),
  },
]

export const router = createBrowserRouter(routes)
