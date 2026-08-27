import { useParams } from "react-router";
import AdminStoryNavigation from "../components/Admin/AdminStoryNavigation";
import ProduceEditor from "../components/Admin/ProduceEditor";

export default function ProduceRoute() {
  const { id } = useParams();

  return (
    <main className="container mx-auto p-6">
      <h1 className="text-2xl font-bold mb-1">Produce Segments</h1>
      <p className="text-sm text-slate-600 mb-4">
        Two English prompts with reference translations, plus the contrastive
        grammar explanation shown after both attempts.
      </p>

      <AdminStoryNavigation storyId={id!} />

      <ProduceEditor storyId={Number(id)} />
    </main>
  );
}
