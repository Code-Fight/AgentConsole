import { Link } from "react-router-dom";

export function ThreadsPage() {
  return (
    <section>
      <h1>Threads</h1>
      <Link to="/threads/thread-1">Open thread-1</Link>
    </section>
  );
}
