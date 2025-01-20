import React from "react";
import Story from "./components/Story.tsx";
import "./styles/AnnotatedText.css";
import "./styles/Story.css";

function App() {
  const storyId = parseInt(
    document.getElementById("root")?.getAttribute("data-story-id") || "0",
  );

  if (!storyId) {
    return <div>Invalid story ID</div>;
  }
  return (
    <div className="App">
      <Story storyId={storyId} />
    </div>
  );
}

export default App;
