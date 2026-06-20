type PageHeroProps = {
  title: string;
  description: string;
  error: string;
};

export function PageHero({ title, description, error }: PageHeroProps) {
  return (
    <section className="page-hero">
      <div>
        <p className="eyebrow">Current module</p>
        <h2>{title}</h2>
        <p className="hero-text">{description}</p>
        {error ? <p className="status-error">{error}</p> : null}
      </div>
    </section>
  );
}
