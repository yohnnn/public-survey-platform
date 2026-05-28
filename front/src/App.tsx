import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { AuthProvider } from "./auth/AuthContext";
import { LiveUpdatesProvider } from "./data/liveUpdates";
import { Layout } from "./components/Layout";
import { ProtectedRoute } from "./components/ProtectedRoute";
import { ToastProvider } from "./components/Toast";
import { AuthPage } from "./pages/AuthPage";
import { CreatePollPage } from "./pages/CreatePollPage";
import { EditProfilePage } from "./pages/EditProfilePage";
import { EditPollPage } from "./pages/EditPollPage";
import { FeedPage } from "./pages/FeedPage";
import { MePage } from "./pages/MePage";
import { NotFoundPage } from "./pages/NotFoundPage";
import { PollPage } from "./pages/PollPage";
import { ProfilePage } from "./pages/ProfilePage";

export function App() {
  return (
    <AuthProvider>
      <LiveUpdatesProvider>
        <ToastProvider>
          <BrowserRouter>
          <Routes>
            <Route element={<Layout />}>
              <Route index element={<FeedPage />} />
              <Route path="feed" element={<Navigate to="/" replace />} />
              <Route path="trending" element={<FeedPage />} />
              <Route path="following" element={<FeedPage />} />
              <Route path="auth" element={<AuthPage />} />
              <Route path="create" element={<ProtectedRoute><CreatePollPage /></ProtectedRoute>} />
              <Route path="me" element={<ProtectedRoute><MePage /></ProtectedRoute>} />
              <Route path="me/edit" element={<ProtectedRoute><EditProfilePage /></ProtectedRoute>} />
              <Route path="poll/:id/edit" element={<ProtectedRoute><EditPollPage /></ProtectedRoute>} />
              <Route path="poll/:id" element={<PollPage />} />
              <Route path="profile/:id" element={<ProfilePage />} />
              <Route path="*" element={<NotFoundPage />} />
            </Route>
          </Routes>
        </BrowserRouter>
        </ToastProvider>
      </LiveUpdatesProvider>
    </AuthProvider>
  );
}
