import { useParams } from "react-router";
import AdminStoryPage from "../components/Admin/AdminStoryPage";
import ProduceEditor from "../components/Admin/ProduceEditor";
import { pageMeta } from "~/lib/pageTitle";

export function meta() {
  return pageMeta("Produce Authoring");
}

export default function ProduceRoute() {
  const { id } = useParams();

  return (
    <AdminStoryPage
      storyId={id!}
      title="Produce Segments"
      description="Two English prompts with reference translations, plus the contrastive grammar explanation shown after both attempts."
    >
      <ProduceEditor storyId={Number(id)} />
    </AdminStoryPage>
  );
}
