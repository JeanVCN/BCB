import './App.css'

function App() {
  return (
    <main className="shell">
      <section className="hero" aria-labelledby="page-title">
        <span className="eyebrow">Big Chat Brasil</span>
        <h1 id="page-title">Conversas que respeitam cada centavo.</h1>
        <p>
          A fundação do BCB está no ar. Cadastro, financeiro e mensagens serão
          adicionados em incrementos verificáveis.
        </p>

        <div className="status" role="status">
          <span aria-hidden="true" />
          Ambiente inicial configurado
        </div>
      </section>
    </main>
  )
}

export default App
