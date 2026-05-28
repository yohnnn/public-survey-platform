import { FormEvent, useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { LoadingState } from "../components/LoadingState";
import { useToast } from "../components/Toast";
import { countries } from "../data/countries";

export function EditProfilePage() {
  const { api, me, setMe, loadMe } = useAuth();
  const toast = useToast();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let active = true;
    void loadMe(true).finally(() => active && setLoading(false));
    return () => {
      active = false;
    };
  }, [loadMe]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!me || busy) return;
    const data = new FormData(event.currentTarget);
    const birthYear = String(data.get("birthYear") || "").trim();
    const payload = {
      email: String(data.get("email") || "").trim(),
      nickname: String(data.get("nickname") || "").trim(),
      country: String(data.get("country") || "").trim(),
      gender: String(data.get("gender") || "").trim(),
      ...(birthYear ? { birthYear: Number(birthYear) } : {}),
    };
    setBusy(true);
    try {
      const response = await api.updateMe(payload);
      setMe(response.user);
      toast("Профиль обновлён.");
      navigate(`/profile/${encodeURIComponent(response.user.id)}`);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Не удалось обновить профиль.", "error");
    } finally {
      setBusy(false);
    }
  }

  if (loading || !me) return <LoadingState title="Редактирование профиля" />;

  return (
    <div className="page-view stack">
      <section className="page-head">
        <div>
          <h1>Редактирование профиля</h1>
        </div>
      </section>

      <section className="card stack profile-edit-form">
        <form className="stack" onSubmit={submit}>
          <label>
            Email
            <input name="email" type="email" defaultValue={me.email || ""} required />
          </label>
          <label>
            Никнейм
            <input name="nickname" defaultValue={me.nickname || ""} required />
          </label>
          <div className="grid-two">
            <label>
              Страна
              <select name="country" defaultValue={me.country || "RU"}>
                {countries.map((c) => (
                  <option key={c.code} value={c.code}>
                    {c.name}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Год рождения
              <input name="birthYear" type="number" min="1900" max="2100" defaultValue={me.birthYear || ""} />
            </label>
          </div>
          <label>
            Пол
            <select name="gender" defaultValue={me.gender || "male"}>
              <option value="male">Мужской</option>
              <option value="female">Женский</option>
            </select>
          </label>
          <div className="actions">
            <button type="submit" disabled={busy}>{busy ? "Сохранение..." : "Сохранить"}</button>
            <Link className="button secondary" to={`/profile/${encodeURIComponent(me.id)}`}>
              Отмена
            </Link>
          </div>
        </form>
      </section>
    </div>
  );
}
