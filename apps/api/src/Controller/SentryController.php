<?php

namespace Omed\CMS\Controller;

use Symfony\Bundle\FrameworkBundle\Controller\AbstractController;
use Symfony\Component\Routing\Attribute\Route;

class SentryController extends AbstractController
{
    #[Route('/sentry-error', name: 'sentry')]
    public function homepage(): void
    {
        throw new \RuntimeException("Unhandled");
    }
}
