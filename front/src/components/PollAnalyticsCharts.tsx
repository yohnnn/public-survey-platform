import {
  Bar,
  BarChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { AgeStat, CountryStat, GenderStat, Poll } from "../types/domain";
import { genderLabel, toCount } from "../utils/format";

interface PollAnalyticsChartsProps {
  poll: Poll;
  countries: CountryStat[];
  gender: GenderStat[];
  age: AgeStat[];
}

export function PollAnalyticsCharts({ poll, countries, gender, age }: PollAnalyticsChartsProps) {
  const total = Number(poll.totalVotes || 0);

  const countryData = collapseTopItems(
    [...countries]
      .map((item) => ({ name: item.country || "—", votes: Number(item.votes || 0) }))
      .filter((item) => item.votes > 0)
      .sort((a, b) => b.votes - a.votes),
    8,
  );

  const genderData = mergeGenderStats(gender)
    .map((item) => ({ name: genderLabel(item.gender), votes: item.votes }))
    .filter((item) => item.votes > 0)
    .sort((a, b) => b.votes - a.votes);

  const ageData = [...age]
    .map((item) => ({ name: item.ageRange || "—", votes: Number(item.votes || 0) }))
    .filter((item) => item.votes > 0)
    .sort((a, b) => b.votes - a.votes);

  const hasDemographics = countryData.length > 0 || genderData.length > 0 || ageData.length > 0;

  if (!total && !hasDemographics) {
    return <div className="chart-empty">Данных пока нет — нужны голоса участников.</div>;
  }

  return (
    <section className="stack">
      <div className="chart-grid">
        <ChartCard title="По полу">
          {genderData.length ? (
            <ResponsiveContainer width="100%" height={Math.max(180, genderData.length * 48)}>
              <BarChart data={genderData} layout="vertical" margin={{ left: 8, right: 16, top: 4, bottom: 4 }}>
                <XAxis type="number" allowDecimals={false} tick={{ fontSize: 12 }} />
                <YAxis type="category" dataKey="name" width={96} tick={{ fontSize: 12 }} />
                <Tooltip formatter={(value) => [toCount(Number(value)), "голосов"]} labelStyle={{ color: "#20231f" }} />
                <Bar dataKey="votes" fill="#226d58" radius={[0, 6, 6, 0]} />
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div className="chart-empty">Нет данных по полу (у голосовавших не указан пол в профиле).</div>
          )}
        </ChartCard>

        <ChartCard title="По странам">
          {countryData.length ? (
            <ResponsiveContainer width="100%" height={240}>
              <BarChart data={countryData} margin={{ left: 0, right: 8, top: 4, bottom: 4 }}>
                <XAxis dataKey="name" tick={{ fontSize: 11 }} interval={0} angle={-25} textAnchor="end" height={60} />
                <YAxis allowDecimals={false} tick={{ fontSize: 12 }} width={36} />
                <Tooltip formatter={(value) => [toCount(Number(value)), "голосов"]} />
                <Bar dataKey="votes" fill="#3d9178" radius={[6, 6, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div className="chart-empty">Нет данных по странам.</div>
          )}
        </ChartCard>

        <ChartCard title="По возрасту">
          {ageData.length ? (
            <ResponsiveContainer width="100%" height={240}>
              <BarChart data={ageData} margin={{ left: 0, right: 8, top: 4, bottom: 4 }}>
                <XAxis dataKey="name" tick={{ fontSize: 12 }} />
                <YAxis allowDecimals={false} tick={{ fontSize: 12 }} width={36} />
                <Tooltip formatter={(value) => [toCount(Number(value)), "голосов"]} />
                <Bar dataKey="votes" fill="#174c3e" radius={[6, 6, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div className="chart-empty">Нет данных по возрасту.</div>
          )}
        </ChartCard>
      </div>
    </section>
  );
}

function ChartCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="chart-card">
      <h4>{title}</h4>
      {children}
    </div>
  );
}

function collapseTopItems(items: { name: string; votes: number }[], limit: number) {
  if (items.length <= limit) return items;
  const top = items.slice(0, limit);
  const restVotes = items.slice(limit).reduce((sum, item) => sum + item.votes, 0);
  if (restVotes > 0) top.push({ name: "Другие", votes: restVotes });
  return top;
}

function mergeGenderStats(items: GenderStat[]): GenderStat[] {
  const merged = new Map<string, number>();
  for (const item of items) {
    const key = normalizeGenderKey(item.gender);
    if (!key) continue;
    merged.set(key, (merged.get(key) || 0) + Number(item.votes || 0));
  }
  return [...merged.entries()].map(([gender, votes]) => ({ gender, votes }));
}

function normalizeGenderKey(value?: string): string {
  const normalized = String(value || "")
    .trim()
    .toLowerCase();
  if (normalized === "m" || normalized === "male" || normalized === "man" || normalized === "мужской") return "male";
  if (normalized === "f" || normalized === "female" || normalized === "woman" || normalized === "женский") return "female";
  return "";
}
