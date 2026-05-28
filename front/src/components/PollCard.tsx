import { Link } from "react-router-dom";
import { useLiveUpdates } from "../data/liveUpdates";
import { tagLabel } from "../data/tags";
import type { FeedItem } from "../types/domain";
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
  const { mergeFeedItem } = useLiveUpdates();
  const live = mergeFeedItem(item);
  const options = live.options || [];
  const totalVotes = Number(live.totalVotes || 0);
  const author = live.author;
  const authorId = author?.id || live.creatorId;

  return (
    <article className="card poll-card">
      <div className="poll-card-header">
        <div>
          <h3>
            <Link to={`/poll/${encodeURIComponent(live.id)}`}>{live.question || "Без вопроса"}</Link>
          </h3>
          <div className="small muted">
            {authorId ? (
              <>
                <Link to={`/profile/${encodeURIComponent(authorId)}`}>{author?.nickname || authorId}</Link>
                {" · "}
                {formatDate(live.createdAt)}
              </>
            ) : (
              formatDate(live.createdAt) || "Автор неизвестен"
            )}
          </div>
        </div>
        <span className="tag">{toCount(totalVotes)} голосов</span>
      </div>
      {live.imageUrl ? <img className="poll-image" src={live.imageUrl} alt="" loading="lazy" /> : null}
      <div className="option-list">
        {options.slice(0, 4).map((option) => (
          <ResultRow key={option.id} label={option.text} votes={option.votesCount} totalVotes={totalVotes} />
        ))}
      </div>
      {(live.tags || []).length ? (
        <div className="footer-actions">
          {(live.tags || []).map((tag) => (
            <Link key={tag} className="tag" to={`/?tags=${encodeURIComponent(tag)}`}>
              {tagLabel(tag)}
            </Link>
          ))}
        </div>
      ) : null}
    </article>
  );
}

export function ResultRow({ label, votes, totalVotes }: { label: string; votes?: number; totalVotes: number }) {
  const count = Math.abs(Number(votes || 0));
  const absTotal = Math.abs(totalVotes);
  const percent = absTotal > 0 ? Math.round((count / absTotal) * 100) : 0;
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
