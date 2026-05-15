export function ErrorState({ message, onRetry }: { message?: string; onRetry?: () => void }) {
  return (
    <section className="page-head">
      <div>
        <h1>Что-то пошло не так</h1>
        <p>{message || "Не удалось загрузить данные."}</p>
      </div>
      {onRetry ? (
        <button type="button" onClick={onRetry}>
          Повторить
        </button>
      ) : null}
    </section>
  );
}
