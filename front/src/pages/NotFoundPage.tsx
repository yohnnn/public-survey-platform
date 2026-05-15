import { Link } from "react-router-dom";

export function NotFoundPage() {
  return (
    <section className="page-head">
      <div>
        <h1>Страница не найдена</h1>
        <p>Такого раздела нет.</p>
      </div>
      <Link className="button" to="/">На главную</Link>
    </section>
  );
}
