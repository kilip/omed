<?php

/*
 * This file is part of the Omed project.
 *
 * (c) Anthonius Munthi <https://itstoni.com>
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

declare(strict_types=1);

namespace Omed\Laravel\User\Http\Controllers;

use Illuminate\Http\Request;
use Omed\Laravel\Core\Http\Controllers\Controller;
use Omed\Laravel\User\Http\Resources\UserResource;
use Omed\Laravel\User\Services\UserManager;

class UserController extends Controller
{
    /**
     * @param UserManager $manager
     *
     * @return \Illuminate\Http\Resources\Json\AnonymousResourceCollection
     */
    public function index(UserManager $manager)
    {
        $pager = $manager->getUserList();
        $collection = UserResource::collection($pager['data']);
        unset($pager['data']);
        $collection->additional($pager);
        return $collection;
    }

    public function show(UserManager $manager, $user)
    {
        $user = $manager->findById($user);
        return new UserResource($user);
    }
}
