export function LoadingState({ title = "Загрузка" }: { title?: string }) {
  return (
    <div className="page-view">
      <section className="page-head">
        <div>
          <h1>{title}</h1>
          <p>Получаем данные...</p>
        </div>
      </section>
      <section className="skeleton-grid">
        <div className="skeleton-card" />
        <div className="skeleton-card" />
        <div className="skeleton-card" />
      </section>
    </div>
  );
}
