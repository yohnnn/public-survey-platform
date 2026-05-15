import { BrowserRouter, Route, Routes } from "react-router-dom";
import { AuthProvider } from "./auth/AuthContext";
import { Layout } from "./components/Layout";
import { ProtectedRoute } from "./components/ProtectedRoute";
import { ToastProvider } from "./components/Toast";
import { AuthPage } from "./pages/AuthPage";
import { CreatePollPage } from "./pages/CreatePollPage";
import { FeedPage } from "./pages/FeedPage";
import { HomePage } from "./pages/HomePage";
import { MePage } from "./pages/MePage";
import { NotFoundPage } from "./pages/NotFoundPage";
import { PollPage } from "./pages/PollPage";
import { ProfilePage } from "./pages/ProfilePage";
import { TagsPage } from "./pages/TagsPage";

export function App() {
  return (
    <AuthProvider>
      <ToastProvider>
        <BrowserRouter>
          <Routes>
            <Route element={<Layout />}>
              <Route index element={<HomePage />} />
              <Route path="auth" element={<AuthPage />} />
              <Route path="feed" element={<FeedPage />} />
              <Route path="trending" element={<FeedPage />} />
              <Route path="following" element={<FeedPage />} />
              <Route path="create" element={<ProtectedRoute><CreatePollPage /></ProtectedRoute>} />
              <Route path="me" element={<ProtectedRoute><MePage /></ProtectedRoute>} />
              <Route path="tags" element={<TagsPage />} />
              <Route path="poll/:id" element={<PollPage />} />
              <Route path="profile/:id" element={<ProfilePage />} />
              <Route path="*" element={<NotFoundPage />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </ToastProvider>
    </AuthProvider>
  );
}
