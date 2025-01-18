import React from "react";
import Story from "./components/Story.tsx";
import "./styles/AnnotatedText.css";
import "./styles/Story.css";

function App() {
  return (
    <div className="App">
      <Story storyId={1} />
    </div>
  );
}

export default App;
