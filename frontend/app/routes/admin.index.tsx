import { useLoaderData, Link } from "react-router";
import { getApiBase, getAdminBase } from "../config";

type StoryListItem = {
  id: number;
  title: string;
  week_wumber: number;
  day_letter: string;
};

export async function loader() {
  const res = await fetch(`${getApiBase()}/stories`);
  const json = await res.json();
  return json.data.stories as StoryListItem[];
}

export default function AdminHome() {
  const stories = useLoaderData() as StoryListItem[];
  return (
    <main className="container mx-auto p-6">
      <h1 className="text-2xl font-bold mb-4">Admin Dashboard</h1>
      <div className="mb-4">
        <Link to="/admin/stories/add" className="text-blue-600">Add New Story</Link>
      </div>
      <ul className="space-y-2">
        {stories.map((s) => (
          <li key={s.id} className="border p-3 rounded">
            <div className="flex justify-between">
              <Link to={`/admin/stories/${s.id}`}>{s.title || `Story #${s.id}`}</Link>
              <div className="text-sm text-gray-600">Week {s.week_wumber}{s.day_letter}</div>
            </div>
          </li>
        ))}
      </ul>
    </main>
  );
}


