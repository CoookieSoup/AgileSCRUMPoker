import { Link } from "react-router-dom";
import { Header } from "../components/Header";

export function NotFoundPage() {
  return (
    <>
      <Header />
      <main className="centered">
        <div className="panel message-panel">
          <h2>Page not found</h2>
          <p>There is nothing at this address.</p>
          <Link to="/" className="button button-accent">
            Back home
          </Link>
        </div>
      </main>
    </>
  );
}
