import { Link } from "react-router-dom";
import { Header } from "../components/Header";
import { PlayingCard } from "../components/PlayingCard";

const FAN = [-30, -20, -10, 0, 10, 20, 30];

const FEATURES = [
  { title: "No sign-up", text: "Create a room, share the link, and start estimating in seconds." },
  { title: "Real-time", text: "Votes, joins and reveals reach everyone instantly over a live connection." },
  { title: "Hidden until reveal", text: "Cards stay secret until the host reveals, so nobody anchors the team." },
  { title: "Ephemeral", text: "Rooms vanish after an hour. No accounts, no data left behind." },
];

const STEPS = [
  "Start a game and copy the invite link.",
  "Teammates open the link and pick a name.",
  "Everyone picks a card; the host reveals and discusses.",
];

export function LandingPage() {
  return (
    <>
      <Header nav />
      <main className="landing">
        <section className="hero">
          <h1>Agile Estimation, Elevated</h1>
          <p>Collaborate, vote, and align your Scrum teams in real-time.</p>
          <Link to="/new" className="button button-primary button-large">
            Start new game
          </Link>
        </section>

        <section className="hero-stage" aria-hidden="true">
          <div className="card-fan">
            {FAN.map((offset, index) => (
              <PlayingCard
                key={offset}
                glow={index % 3 === 0}
                style={{ transform: `translateX(${offset * 7}px) rotate(${offset / 3}deg)`, zIndex: 10 - Math.abs(offset) / 10 }}
              />
            ))}
          </div>
        </section>

        <section id="features" className="feature-grid">
          {FEATURES.map((feature) => (
            <article key={feature.title} className="panel feature">
              <h3>{feature.title}</h3>
              <p>{feature.text}</p>
            </article>
          ))}
        </section>

        <section id="how" className="how">
          <h2>How it works</h2>
          <ol>
            {STEPS.map((step) => (
              <li key={step}>{step}</li>
            ))}
          </ol>
        </section>
      </main>
    </>
  );
}
