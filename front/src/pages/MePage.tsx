import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { LoadingState } from "../components/LoadingState";

export function MePage() {
  const { me, loadMe } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    let active = true;
    void (async () => {
      const user = me || (await loadMe(true));
      if (!active) return;
      if (user?.id) {
        navigate(`/profile/${encodeURIComponent(user.id)}`, { replace: true });
        return;
      }
      navigate("/auth", { replace: true });
    })();
    return () => {
      active = false;
    };
  }, [loadMe, me, navigate]);

  return <LoadingState title="Профиль" />;
}
