<script>
  import Battleship from "$lib/assets/Battleshiper.svg"
  import Wave from "$lib/assets/Wave.svg"
  import { cn } from "$lib/utils";

  /** @type {string} */
  export let className;
</script>

<div class={cn("relative flex justify-center", className)}>
  <img class="battleship w-11/12" alt="Battleship" src={Battleship} />
  <div class="wave-container absolute bottom-0 w-full h-[20%] overflow-hidden">
    <img class="wave absolute top-0 left-0" alt="wave" src={Wave} />
    <img class="wave delay-1 absolute top-0 left-0" alt="wave" src={Wave} />
    <img class="wave delay-2 absolute top-0 left-0" alt="wave" src={Wave} />
    <img class="wave delay-3 absolute top-0 left-0" alt="wave" src={Wave} />
  </div>
</div>

<style>

  .battleship {
    opacity: 0;
    animation: show-battleship 4s ease-out forwards;
  }

  .wave-container {
    mask-image: radial-gradient(circle, rgba(0, 0, 0, 1) 60%, rgba(0, 0, 0, 0) 100%);
    -webkit-mask-image: radial-gradient(circle, rgba(0, 0, 0, 1) 60%, rgba(0, 0, 0, 0) 100%);
    mask-size: cover;
    -webkit-mask-size: cover;
  }

  .wave {
    @apply w-[200%];
    opacity: 0;
    animation-delay: 0s;
    animation: 
      show-wave 3s ease forwards,
      move-wave 12s ease-in-out infinite;
  }
  .wave.delay-1 {
    animation-delay: 3s;
  }
  .wave.delay-2 {
    animation-delay: 6s;
  }
  .wave.delay-3 {
    animation-delay: 9s;
  }

  @keyframes show-battleship {
    0% {
      opacity: 0;
      transform: translate(20%, -10%);
    }
    100% {
      opacity: 1;
      transform: translate(0%, 0%);
    }
  }

  @keyframes show-wave {
    0% {
      opacity: 0;
    }
    100% {
      opacity: 1;
    }
  }

  @keyframes move-wave {
    0% {
      transform: translateX(-100%);
    }
    100% {
      transform: translateX(100%);
    }
  }
</style>