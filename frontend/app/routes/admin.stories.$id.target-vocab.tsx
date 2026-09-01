import { useParams } from "react-router";
import AdminStoryPage from "../components/Admin/AdminStoryPage";
import TargetVocabEditor from "../components/Admin/TargetVocabEditor";
import { pageMeta } from "~/lib/pageTitle";

export function meta() {
  return pageMeta("Target Vocabulary");
}

export default function TargetVocabRoute() {
  const { id } = useParams();

  return (
    <AdminStoryPage
      storyId={id!}
      title="Target Vocabulary"
      description="The five words the Identify and Recall phases are built on."
    >
      <TargetVocabEditor storyId={Number(id)} />
    </AdminStoryPage>
  );
}
