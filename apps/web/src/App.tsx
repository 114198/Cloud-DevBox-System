import { BrowserRouter, Routes, Route, Navigate, useLocation } from 'react-router-dom'
import MainLayout from './layouts/MainLayout'
import Dashboard from './pages/Dashboard'
import Environments from './pages/Environments'
import EnvironmentDetail from './pages/EnvironmentDetail'
import EnvironmentConfig from './pages/EnvironmentConfig'
import Templates from './pages/Templates'
import Projects from './pages/Projects'
import Settings from './pages/Settings'
import Login from './pages/Login'
import Register from './pages/Register'
import OAuthCallback from './pages/OAuthCallback'
import GitRepositories from './pages/GitRepositories'
import GitOperations from './pages/GitOperations'
import GitOAuthCallback from './pages/GitOAuthCallback'
import Collaboration from './pages/Collaboration'
import Meeting from './pages/Meeting'
import Deployments from './pages/Deployments'
import DeploymentHistory from './pages/DeploymentHistory'
import DeploymentLogs from './pages/DeploymentLogs'
import Billing from './pages/Billing'
import Plans from './pages/Plans'
import Backup from './pages/Backup'
import { useAuthStore } from './stores/auth'

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
  const location = useLocation()
  
  if (!isAuthenticated) {
    // Save the attempted URL for redirecting after login
    return <Navigate to="/login" state={{ from: location.pathname }} replace />
  }
  
  return <>{children}</>
}

function PublicRoute({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
  
  // Redirect to dashboard if already authenticated
  if (isAuthenticated) {
    return <Navigate to="/" replace />
  }
  
  return <>{children}</>
}

function App() {
  return (
    <BrowserRouter>
      <Routes>
        {/* Public routes */}
        <Route 
          path="/login" 
          element={
            <PublicRoute>
              <Login />
            </PublicRoute>
          } 
        />
        <Route 
          path="/register" 
          element={
            <PublicRoute>
              <Register />
            </PublicRoute>
          } 
        />
        
        {/* OAuth callback route */}
        <Route path="/auth/callback/:provider" element={<OAuthCallback />} />
        
        {/* Git OAuth callback route */}
        <Route path="/git/callback/:provider" element={<GitOAuthCallback />} />
        
        {/* Protected routes */}
        <Route
          path="/"
          element={
            <PrivateRoute>
              <MainLayout />
            </PrivateRoute>
          }
        >
          <Route index element={<Dashboard />} />
          <Route path="environments" element={<Environments />} />
          <Route path="environments/:id" element={<EnvironmentDetail />} />
          <Route path="environments/:id/config" element={<EnvironmentConfig />} />
          <Route path="environments/:id/collaborate" element={<Collaboration />} />
          <Route path="templates" element={<Templates />} />
          <Route path="projects" element={<Projects />} />
          <Route path="git" element={<GitRepositories />} />
          <Route path="git/:id" element={<GitOperations />} />
          <Route path="deployments" element={<Deployments />} />
          <Route path="deployments/:id/history" element={<DeploymentHistory />} />
          <Route path="deployments/:id/logs" element={<DeploymentLogs />} />
          <Route path="billing" element={<Billing />} />
          <Route path="billing/plans" element={<Plans />} />
          <Route path="backup" element={<Backup />} />
          <Route path="settings" element={<Settings />} />
        </Route>
        
        {/* Collaboration join route (standalone page) */}
        <Route
          path="/collaboration/join/:token"
          element={
            <PrivateRoute>
              <Collaboration />
            </PrivateRoute>
          }
        />
        
        {/* Meeting routes (standalone pages) */}
        <Route
          path="/meeting/:id"
          element={
            <PrivateRoute>
              <Meeting />
            </PrivateRoute>
          }
        />
        
        {/* Catch all - redirect to dashboard */}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App
