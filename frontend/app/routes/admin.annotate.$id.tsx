// Admin annotator route. This mirrors /admin/stories/{id}/annotate Go template but uses SPA.
import { useParams } from "react-router";
import Story from "../components/Annotator/Story";

export default function AdminAnnotateRoute() {
  const params = useParams();
  const id = Number(params.id);
  if (!id) return <div>Invalid story ID</div>;
  return <Story storyId={id} />;
}


