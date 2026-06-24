# Critérios de aceite do BCB

Este documento converte o escopo aprovado em comportamentos verificáveis. Ele
não define endpoints, banco de dados ou estrutura de código.

## 1. Ambiente e entrega

- [x] Todo o ambiente necessário pode ser iniciado com Docker.
- [x] Backend, frontend, PostgreSQL e Redis conseguem se comunicar no ambiente
  documentado.
- [x] Uma instalação limpa possui um caminho documentado para criar o primeiro
  administrador.
- [x] Dados persistidos no PostgreSQL sobrevivem ao reinício dos serviços.
- [ ] A aplicação apresenta erros de configuração sem expor segredos.

## 2. Cadastro e ativação

- [x] Uma empresa consegue se cadastrar como PF/CPF ou PJ/CNPJ.
- [x] CPF/CNPJ duplicado não cria uma segunda conta.
- [x] A senha aceita entre 9 e 128 caracteres e não é truncada.
- [x] A senha exige letras, números e caractere especial, conforme decisão do
  projeto.
- [x] Senhas conhecidas como comuns ou comprometidas são rejeitadas.
- [x] A senha nunca é persistida ou registrada em texto puro.
- [x] Uma empresa recém-cadastrada permanece inativa.
- [x] Uma empresa inativa não consegue autenticar.
- [x] Uma empresa inativa não consegue enviar mensagens.
- [x] Um administrador consegue consultar cadastros aguardando ativação.
- [x] Somente um administrador consegue ativar ou inativar uma empresa.
- [x] Ativação e inativação registram ator, instante e mudança realizada.

## 3. Autenticação e autorização

- [x] Uma empresa ativa autentica com documento e senha corretos.
- [x] Credenciais inválidas retornam uma resposta que não revela qual campo
  estava incorreto.
- [x] Tentativas de autenticação são limitadas progressivamente.
- [x] Um cliente acessa somente suas conversas e mensagens.
- [x] Um cliente acessa somente seus próprios dados financeiros.
- [x] Um usuário `client` não executa operações exclusivas de `admin`.
- [x] Uma sessão inválida ou expirada não acessa recursos protegidos.

## 4. Planos e administração financeira

- [x] Uma empresa possui exatamente um plano vigente: pré-pago ou pós-pago.
- [x] Administrador consegue adicionar crédito ao pré-pago.
- [x] Administrador consegue alterar o limite total do pós-pago sem apagar seu
  consumo acumulado.
- [x] Saldo, limite total, consumo e disponibilidade são apresentados de forma
  coerente com o plano.
- [x] Toda alteração financeira cria histórico rastreável.
- [x] Operações financeiras concorrentes não produzem saldo negativo, consumo
  perdido ou cobrança duplicada.
- [x] Indisponibilidade do lock distribuído não permite prosseguir de forma
  insegura com uma mutação financeira.

## 5. Solicitação de mudança de plano

- [x] Cliente pré-pago somente solicita mudança com saldo igual a zero.
- [x] Cliente pós-pago somente solicita mudança com consumo igual a zero.
- [x] Um cliente não cria duas solicitações pendentes.
- [x] O cliente pode cancelar sua solicitação pendente.
- [x] O painel administrativo exibe contador e lista de solicitações pendentes.
- [x] Um administrador pode aprovar ou rejeitar uma solicitação.
- [x] A aprovação revalida autorização e condição financeira.
- [x] Uma aprovação inválida não altera o plano.
- [x] `pending`, `approved`, `rejected` e `cancelled` são transições
  controladas e auditáveis.

## 6. Conversas

- [x] Uma empresa ativa consegue cadastrar conversa com nome e telefone do
  cliente final.
- [x] A empresa visualiza somente suas conversas.
- [x] A lista apresenta destinatário e resumo da atividade mais recente.
- [x] Selecionar uma conversa carrega seu histórico.
- [x] Estados vazio, carregando e erro possuem apresentação apropriada.

## 7. Envio e financeiro da mensagem

- [x] A empresa escolhe SMS ou WhatsApp e prioridade normal ou urgente.
- [x] Mensagem normal custa R$ 0,25 e urgente custa R$ 0,50.
- [x] Pré-pago sem saldo suficiente não é cobrado nem enfileirado.
- [x] Pós-pago sem limite disponível não é consumido nem enfileirado.
- [x] Uma mensagem válida registra cliente, conversa, destinatário, canal,
  conteúdo, prioridade, custo e instante.
- [x] Cobrança, registro da mensagem e encaminhamento para processamento não
  deixam estado financeiro parcial observável.
- [x] Repetir uma mesma solicitação identificada não gera mensagem ou cobrança
  duplicada.

## 8. Fila, worker, retry e estorno

- [x] Mensagem aceita inicia como `queued` e passa por `processing`.
- [x] O worker processa uma mensagem no máximo uma vez simultaneamente.
- [x] A fila simples preserva a ordenação definida para a primeira versão.
- [x] Uma falha transitória realiza no máximo três retries depois da tentativa
  inicial.
- [ ] Retries usam backoff exponencial com jitter e configuração testável.
- [x] Falha permanente não é repetida.
- [x] Sucesso termina em `sent` e não gera estorno.
- [x] Esgotamento dos retries termina em `failed`.
- [x] Uma mensagem definitivamente `failed` gera exatamente um estorno ligado
  à transação original.
- [x] Retry, processamento e estorno permanecem idempotentes após reinício ou
  execução concorrente.

## 9. Frontend e integração

- [x] O fluxo cadastro → ativação → login → conversa → envio pode ser
  demonstrado de ponta a ponta.
- [x] O frontend apresenta loading, vazio, erro e sucesso nas operações
  principais.
- [x] A lista e o histórico refletem o estado persistido no backend.
- [x] O frontend acompanha `queued`, `processing`, `sent` e `failed` sem
  apresentar falso sucesso.
- [x] Falhas de saldo, limite, validação, autorização e comunicação possuem
  mensagens compreensíveis.
- [x] O layout funciona de maneira utilizável em mobile e desktop.
- [ ] Recarregar a página não perde dados persistidos e trata a sessão
  corretamente.

## 10. Auditoria e segurança operacional

- [x] Operações administrativas e financeiras relevantes possuem auditoria.
- [x] Logs não expõem senha, token ou outros segredos.
- [ ] Erros internos não devolvem detalhes sensíveis ao frontend.
- [x] Datas e valores financeiros mantêm representação consistente entre as
  camadas.

## Fora do primeiro escopo

- Integração real com SMS ou WhatsApp.
- Mensagens inbound/recebidas de provedor externo.
- Estados `delivered` e `read`.
- RabbitMQ.
- E-mail, push ou WebSocket para notificações administrativas.
- Reset mensal automatizado, paginação e busca avançada.

## Definição de pronto desta fase

Os comportamentos deste documento estão modelados em `domain-model.md` e
`api-contracts.md`. Novas ambiguidades descobertas durante a implementação devem
ser registradas antes de alterar os contratos.
