import { DECK } from "../api/types";

interface CardDeckProps {
  selected?: string;
  disabled: boolean;
  onPick: (card: string) => void;
}

export function CardDeck({ selected, disabled, onPick }: CardDeckProps) {
  return (
    <div className="card-deck" role="radiogroup" aria-label="Your estimate">
      {DECK.map((card) => (
        <button
          key={card}
          role="radio"
          aria-checked={selected === card}
          className={`deck-card ${selected === card ? "selected" : ""}`}
          disabled={disabled}
          onClick={() => onPick(card)}
        >
          {card}
        </button>
      ))}
    </div>
  );
}
