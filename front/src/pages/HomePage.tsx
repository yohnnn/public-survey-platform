import { BarChart3, Radio, UsersRound } from "lucide-react";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { PollList } from "../components/PollCard";
import type { FeedItem } from "../types/domain";
import { toCount } from "../utils/format";

export function HomePage() {
  const { api, isAuthenticated, me } = useAuth();
  const [feed, setFeed] = useState<FeedItem[]>([]);
  const [trendingCount, setTrendingCount] = useState(0);

  useEffect(() => {
    let active = true;
    Promise.all([api.feed("/v1/feed", "limit=3"), api.feed("/v1/feed/trending", "limit=3")])
      .then(([feedResponse, trendingResponse]) => {
        if (!active) return;
        setFeed(feedResponse.items || []);
        setTrendingCount(trendingResponse.items?.length || 0);
      })
      .catch(() => undefined);
    return () => {
      active = false;
    };
  }, [api]);

  return (
    <div className="page-view">
      <section className="hero">
        <div className="hero-copy">
          <span className="eyebrow">Public Survey</span>
          <h1>Платформа для публичных опросов</h1>
          <p>Публикуйте вопросы, голосуйте, сравнивайте мнения и смотрите аналитику по аудитории в одном веб-интерфейсе.</p>
          <div className="hero-actions">
            <Link className="button" to={isAuthenticated ? "/create" : "/auth?mode=register"}>
              {isAuthenticated ? "Создать опрос" : "Начать"}
            </Link>
            <Link className="button secondary" to="/feed">
              Смотреть ленту
            </Link>
          </div>
        </div>
        <div className="hero-panel">
          <div className="stat-tile">
            <Radio size={20} />
            <strong>{toCount(feed.length)}</strong>
            <span>новых опросов</span>
          </div>
          <div className="stat-tile">
            <BarChart3 size={20} />
            <strong>{toCount(trendingCount)}</strong>
            <span>в трендах</span>
          </div>
          <div className="stat-tile">
            <UsersRound size={20} />
            <strong>{me ? "1" : "0"}</strong>
            <span>активная сессия</span>
          </div>
        </div>
      </section>

      <section className="grid-three">
        <article className="card feature-card">
          <h3>Участие</h3>
          <p className="muted">Пользователь быстро выбирает вариант и сразу видит распределение ответов.</p>
        </article>
        <article className="card feature-card">
          <h3>Публикация</h3>
          <p className="muted">Автор создает вопрос, добавляет варианты, теги и изображение.</p>
        </article>
        <article className="card feature-card">
          <h3>Аналитика</h3>
          <p className="muted">Результаты доступны по вариантам ответа и демографическим признакам.</p>
        </article>
      </section>

      <section className="stack">
        <div className="section-head">
          <h2>Последние публикации</h2>
          <Link to="/feed">Все опросы</Link>
        </div>
        <PollList items={feed} />
      </section>
    </div>
  );
}
