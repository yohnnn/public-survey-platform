import {
  Bar,
  BarChart,
  Cell,
  Legend,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { AgeStat, CountryStat, GenderStat, Poll, PollAnalytics } from "../types/domain";
import { toCount } from "../utils/format";

const CHART_COLORS = ["#226d58", "#3d9178", "#5aad93", "#7bc4ad", "#9ad9c4", "#174c3e", "#688f7f", "#a8c9bc"];

interface PollAnalyticsChartsProps {
  poll: Poll;
  analytics?: PollAnalytics;
  countries: CountryStat[];
  gender: GenderStat[];
  age: AgeStat[];
}

export function PollAnalyticsCharts({ poll, analytics, countries, gender, age }: PollAnalyticsChartsProps) {
  const total = Number(poll.totalVotes || 0) || analytics?.totalVotes || 0;

  const optionData = (poll.options || []).map((option) => ({
    name: option.text || option.id,
    votes: Number(option.votesCount || 0),
  }));

  const countryData = collapseTopItems(
    [...countries].sort((a, b) => b.votes - a.votes).map((item) => ({ name: item.country || "—", votes: item.votes })),
    8,
  );

  const genderData = [...gender]
    .sort((a, b) => b.votes - a.votes)
    .map((item) => ({ name: formatGender(item.gender), votes: item.votes }));

  const ageData = [...age]
    .sort((a, b) => b.votes - a.votes)
    .map((item) => ({ name: item.ageRange || "—", votes: item.votes }));

  if (!total) {
    return <div className="chart-empty">Данных пока нет — нужны голоса участников.</div>;
  }

  return (
    <section className="stack">
      <div className="chart-grid">
        <ChartCard title="По вариантам ответа">
          {optionData.length ? (
            <ResponsiveContainer width="100%" height={Math.max(160, optionData.length * 44)}>
              <BarChart data={optionData} layout="vertical" margin={{ left: 8, right: 16, top: 4, bottom: 4 }}>
                <XAxis type="number" allowDecimals={false} tick={{ fontSize: 12 }} />
                <YAxis type="category" dataKey="name" width={110} tick={{ fontSize: 12 }} />
                <Tooltip formatter={(value) => [toCount(Number(value)), "голосов"]} labelStyle={{ color: "#20231f" }} />
                <Bar dataKey="votes" fill="#226d58" radius={[0, 6, 6, 0]} />
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div className="chart-empty">Нет данных по вариантам.</div>
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

        <ChartCard title="По полу">
          {genderData.length ? (
            <ResponsiveContainer width="100%" height={240}>
              <PieChart>
                <Pie data={genderData} dataKey="votes" nameKey="name" innerRadius={52} outerRadius={88} paddingAngle={2}>
                  {genderData.map((entry, index) => (
                    <Cell key={entry.name} fill={CHART_COLORS[index % CHART_COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip formatter={(value) => [toCount(Number(value)), "голосов"]} />
                <Legend />
              </PieChart>
            </ResponsiveContainer>
          ) : (
            <div className="chart-empty">Нет данных по полу.</div>
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

function formatGender(value: string): string {
  const normalized = value.trim().toLowerCase();
  if (normalized === "male" || normalized === "m") return "Мужской";
  if (normalized === "female" || normalized === "f") return "Женский";
  if (!normalized) return "—";
  return value;
}
