import { useParams } from "react-router";
import AdminStoryNavigation from "../components/Admin/AdminStoryNavigation";
import TargetVocabEditor from "../components/Admin/TargetVocabEditor";

export default function TargetVocabRoute() {
  const { id } = useParams();

  return (
    <main className="container mx-auto p-6">
      <h1 className="text-2xl font-bold mb-1">Target Vocabulary</h1>
      <p className="text-sm text-slate-600 mb-4">
        The five words the Identify and Recall phases are built on.
      </p>

      <AdminStoryNavigation storyId={id!} />

      <TargetVocabEditor storyId={Number(id)} />
    </main>
  );
}
