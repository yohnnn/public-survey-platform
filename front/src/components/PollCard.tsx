import { Link } from "react-router-dom";
import type { FeedItem, PollOption } from "../types/domain";
import { formatDate, toCount } from "../utils/format";

export function PollList({ items }: { items: FeedItem[] }) {
  if (!items.length) return <div className="empty">Пока здесь нет опросов.</div>;
  return (
    <div className="feed-list">
      {items.map((item) => (
        <PollCard key={item.id} item={item} />
      ))}
    </div>
  );
}

export function PollCard({ item }: { item: FeedItem }) {
  const options = item.options || [];
  const totalVotes = Number(item.totalVotes || 0);
  const leader = options.reduce<PollOption | null>((best, option) => (Number(option.votesCount || 0) > Number(best?.votesCount || 0) ? option : best), options[0] || null);
  const author = item.author;
  const authorId = author?.id || item.creatorId;

  return (
    <article className="card poll-card">
      <div className="poll-card-header">
        <div>
          <h3>
            <Link to={`/poll/${encodeURIComponent(item.id)}`}>{item.question || "Без вопроса"}</Link>
          </h3>
          <div className="small muted">
            {authorId ? (
              <>
                Автор: <Link to={`/profile/${encodeURIComponent(authorId)}`}>{author?.nickname || authorId}</Link>
              </>
            ) : (
              "Автор неизвестен"
            )}
          </div>
        </div>
        <span className="chip">{formatDate(item.createdAt)}</span>
      </div>
      {item.imageUrl ? <img className="poll-image" src={item.imageUrl} alt="" loading="lazy" /> : null}
      <div className="option-list">
        {options.slice(0, 4).map((option) => (
          <ResultRow key={option.id} label={option.text} votes={option.votesCount} totalVotes={totalVotes} />
        ))}
      </div>
      <div className="footer-actions">
        <span className="tag">{toCount(totalVotes)} голосов</span>
        {leader ? <span className="chip">Лидер: {leader.text || "-"}</span> : null}
        {(item.tags || []).map((tag) => (
          <Link key={tag} className="tag" to={`/feed?tags=${encodeURIComponent(tag)}`}>
            {tag}
          </Link>
        ))}
        <Link className="button secondary compact" to={`/poll/${encodeURIComponent(item.id)}`}>
          Открыть
        </Link>
      </div>
    </article>
  );
}

export function ResultRow({ label, votes, totalVotes }: { label: string; votes?: number; totalVotes: number }) {
  const percent = totalVotes > 0 ? Math.round((Number(votes || 0) / totalVotes) * 100) : 0;
  return (
    <div className="option-row">
      <span>{label || "-"}</span>
      <div className="bar" aria-hidden="true">
        <span style={{ width: `${percent}%` }} />
      </div>
      <strong>{toCount(votes)}</strong>
    </div>
  );
}
