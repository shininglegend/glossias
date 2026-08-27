import { useParams } from "react-router";
import AdminStoryNavigation from "../components/Admin/AdminStoryNavigation";
import RecallEditor from "../components/Admin/RecallEditor";

export default function RecallRoute() {
  const { id } = useParams();

  return (
    <main className="container mx-auto p-6">
      <h1 className="text-2xl font-bold mb-1">Recall Sentences</h1>
      <p className="text-sm text-slate-600 mb-4">
        Five sentences in story order, one per target word. Students see them
        shuffled and reconstruct this order.
      </p>

      <AdminStoryNavigation storyId={id!} />

      <RecallEditor storyId={Number(id)} />
    </main>
  );
}
