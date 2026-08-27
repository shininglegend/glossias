import { useParams } from "react-router";
import AdminStoryPage from "../components/Admin/AdminStoryPage";
import RecallEditor from "../components/Admin/RecallEditor";
import { pageMeta } from "~/lib/pageTitle";

export function meta() {
  return pageMeta("Recall Authoring");
}

export default function RecallRoute() {
  const { id } = useParams();

  return (
    <AdminStoryPage
      storyId={id!}
      title="Recall Sentences"
      description="Five sentences in story order, one per target word. Students see them shuffled and reconstruct this order."
    >
      <RecallEditor storyId={Number(id)} />
    </AdminStoryPage>
  );
}
